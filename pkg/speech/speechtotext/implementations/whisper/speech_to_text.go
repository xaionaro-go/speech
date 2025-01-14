package whisper

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/facebookincubator/go-belt"
	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/lazybeaver/entropy"
	"github.com/mutablelogic/go-whisper/pkg/schema"
	"github.com/mutablelogic/go-whisper/sys/whisper"
	"github.com/xaionaro-go/audio/pkg/audio"
	"github.com/xaionaro-go/audio/pkg/audio/resampler"
	"github.com/xaionaro-go/audio/pkg/vad"
	"github.com/xaionaro-go/observability"
	"github.com/xaionaro-go/speech/pkg/speech"
	"github.com/xaionaro-go/speech/pkg/speech/speechtotext/implementations/whisper/types"
	"github.com/xaionaro-go/xsync"
)

// #cgo pkg-config: libwhisper
// #cgo linux pkg-config: libwhisper-linux
// #cgo darwin pkg-config: libwhisper-darwin
import "C"

const (
	BufferLimit                              = 120 * time.Second
	GapToCommit                              = 2 * time.Second
	DiscardIfNoUsefulSegmentsIterations      = 4
	DiscardFromSingleIterationIfBufferBigger = 10 * time.Second
	IterationInterval                        = time.Second
	PreserveHeadingDuration                  = time.Second
	VADMinVoiceDuration                      = 150 * time.Millisecond

	EntropyMin            = 3.63
	EntropyDetectorLenMin = 80
)

type SpeechToText struct {
	xsync.Mutex
	Context  *whisper.Context
	Out      chan *speech.Transcript
	Received *schema.Transcription
	Params   whisper.FullParams

	NextBuffer       []byte
	CommittingBuffer []byte
	TempBuffer       []byte

	CommittingPosBytes uint64

	IsFirstSpeakerSpeaking bool
	CommitAudioError       error

	CancelFunc context.CancelFunc

	Iterations                 uint
	NoUsefulSegmentsIterations uint
	ModelHash                  [sha1.Size]byte

	VAD          vad.VAD
	VADThreshold float64
	VADBuffer    []byte

	VADCacheVoiceFoundAt time.Duration

	LastSegmentString  string
	LastSegmentStartTS time.Duration
	LastSegmentEndTS   time.Duration
}

var _ speech.ToText = (*SpeechToText)(nil)

func New(
	ctx context.Context,
	modelBytes []byte,
	language speech.Language,
	samplingStrategy types.SamplingStrategy,
	shouldTranslate bool,
	alignmentAheadPreset types.AlignmentAheadsPreset,
	vadThreshold float64,
	opts ...Option,
) (*SpeechToText, error) {
	cfg := Options(opts).config()
	params := whisper.DefaultContextParams()
	if cfg.UseGPU != nil {
		params.SetUseGpu(*cfg.UseGPU)
	}
	if cfg.GPUDeviceID != nil {
		params.SetGpuDevice(*cfg.GPUDeviceID)
	}
	if cfg.FlashAttn != nil {
		params.SetFlashAttn(*cfg.FlashAttn)
	}
	params.SetTokenTimestamps(false)
	params.SetDTWAheadsPreset(whisper.AlignmentAheadsPreset(alignmentAheadPreset))
	whisper.Whisper_log_set(func(level whisper.LogLevel, text string) {
		logger.FromCtx(ctx).Log(logLevelFromWhisper(level), text)
	})

	h := sha1.Sum(modelBytes)
	logger.Debugf(ctx, "model SHA1: %X", h)

	stt := &SpeechToText{
		Context:   whisper.Whisper_init_from_buffer_with_params(modelBytes, params),
		Params:    whisper.DefaultFullParams(samplingStrategy.ToWhisper()),
		Received:  &schema.Transcription{},
		ModelHash: h,

		VADThreshold: vadThreshold,

		IsFirstSpeakerSpeaking: true,
	}

	if vadThreshold > 0 {
		var err error
		stt.VAD, err = stt.newVAD(ctx)
		if err != nil {
			return nil, ErrInitVAD{Err: err}
		}
	}

	if shouldTranslate {
		if !whisper.Whisper_is_multilingual(stt.Context) {
			return nil, ErrModelCannotTranslate{}
		}
	}

	lang := LanguageToWhisper(language)
	logger.Infof(ctx, "language: '%v'; shouldTranslate: %v", lang, shouldTranslate)

	stt.Params.SetTranslate(shouldTranslate)
	stt.Params.SetDiarize(true)
	stt.Params.SetTokenTimestamps(true)
	stt.Params.SetLanguage(lang)

	stt.Params.SetAbortCallback(stt.Context, func() bool {
		select {
		case <-ctx.Done():
			return true
		default:
			return false
		}
	})

	ctx, cancelFn := context.WithCancel(ctx)
	stt.launchProcessingLoop(ctx)
	stt.CancelFunc = cancelFn
	return stt, nil
}

func (stt *SpeechToText) isLikelyHallucination(
	ctx context.Context,
	s *whisper.Segment,
) bool {
	t0 := strings.Trim(s.Text, " ")
	t1 := strings.ReplaceAll(t0, "!", "")
	t1 = strings.ReplaceAll(t1, ".", "")
	t1 = strings.ReplaceAll(t1, "-", "")
	t1 = strings.Trim(t1, " ")
	switch {
	case bytes.Equal(stt.ModelHash[:], ModelHashMedium[:]):
		logger.Tracef(ctx, "hallucination check for medium")
		switch t1 {
		case "Thank you for watching", "Thanks for watching",
			"Thank you for watching Please subscribe to my channel",
			"Thank you",
			"Bye":
			return true
		}

	case bytes.Equal(stt.ModelHash[:], ModelHashLargeV3[:]):
		logger.Tracef(ctx, "hallucination check for large-v3")
		switch t0 {
		case "0.", "0.5.", "0.001.",
			"you",
			"Oh!",
			"Pew.",
			"So,",
			"Thanks.",
			"- What?",
			//"- Mm-hmm.",
			"Hello everyone, welcome to my channel.",
			//"Hi, everyone.",
			"The next day",
			"So, that's it.",
			"So, let's do that.",
			"So, let's do this.",
			"So, let's do it.",
			"Well, I'm going to do it.",
			"So, let's go ahead and do that.",
			"I'm going to go ahead and do that.",
			"So, I'm going to go ahead and do that.",
			"So, we're going to do the same thing.",
			"So, we're going to do that.",
			"I'm going to...",
			"So, we have the following.",
			"I don't know what to do.",
			"We don't know about the fill of our 20 pairs, but it's a big one.",
			"We have 15 minutes left.",
			"I'll be back.",
			"I'll be back in a minute.",
			"I'll be right back.",
			"I'll go and get the baby.",
			"Sleep.",
			"I'm going to bed.",
			"I'm going to sleep.",
			"I'm going to go to sleep",
			"I'm going to go and get some water.",
			"I'll be waiting for you at the station.",
			"So, we have a function called,",
			"So, we have a function called the,",
			"So, we're going to do a little bit of a loop.",
			"All right.",
			"I'll go and get the money.",
			"I love you.",
			"So, what is the relationship between the two?",
			"You're welcome.",
			"Shit.",
			"Amen.",
			"what the hell is that sound?",
			"I'm not a doctor.",
			"You're going to be a good boy.",
			"let's go to the bathroom",
			"I'm sorry. I'll go to the bathroom.",
			"I'm going to the bathroom. I'll be right back.",
			"I'm going to the hospital.",
			"I'm going to the hospital. I'll be there in a minute.",
			"I'm going to make a new one.",
			"I'm going to write a new one.",
			"I'm going to write this.",
			`"I'm sorry.`,
			"I'm sorry, I didn't mean to.",
			"I'm sorry, I didn't mean to hurt you.",
			"I'm sorry. I'm sorry.",
			"I'm sorry, I'm sorry.",
			"I'm sorry, I'm sorry. I'm sorry.",
			"I'm sorry. I'm sorry. I'm sorry.":
			return true
		}
		switch t1 {
		case "Thank you for watching", "Thanks for watching",
			"Thank you for watching Please subscribe to my channel",
			"Thank you",
			"I'm sorry",
			"Bye",
			"Subtitles by the Amaraorg community",
			"Okay",
			"The end",
			"The End",
			"THE END",
			"I'll go to the bathroom",
			"I'm going to the bathroom",
			"":
			return true
		}

		switch {
		case strings.Contains(s.Text, "So, this is the first step"):
			return true
		case strings.Contains(s.Text, "So, we have a function of 0.001"):
			return true
		case strings.HasPrefix(t0, `"`) && strings.HasSuffix(t0, `"`):
			return true
		case strings.HasPrefix(t0, "End of"):
			return true
		}

		if len(t0) > EntropyDetectorLenMin {
			// Examples of what we are suppressing here:
			// * "So, I'm going to write the value of the value of the value of the value of the value of"
			entropy, err := entropy.Shannon(string(s.Text)[30:])
			if err != nil {
				logger.Errorf(ctx, "unable to calculate shannon entropy: %v", err)
				return false
			}

			// Entropy:
			// * "ue of the value of the value of the value of the value of" -> 3.176

			if entropy < EntropyMin {
				logger.Tracef(ctx, "entropy is too low, assuming '%s' is a hallucination: %f < %f", s.Text, entropy, EntropyMin)
				return true
			}
		}
	}

	return false
}

func isHangingSegment(s *whisper.Segment) bool {
	// Sometimes Whisper goes crazy and hangs while processing a specific audio,
	// in this case it returns a lot of exclamation marks and nothing else

	for _, token := range s.Tokens {
		if token.Text != "!" {
			return false
		}
	}
	return true
}

func (stt *SpeechToText) launchProcessingLoop(ctx context.Context) {
	stt.Out = make(chan *speech.Transcript, 1024)
	observability.Go(ctx, func() {
		defer func() {
			close(stt.Out)
			whisper.Whisper_free(stt.Context)
			stt.Context = nil
		}()
		stt.processingLoop(ctx)
	})
}

func (stt *SpeechToText) processingLoop(ctx context.Context) {
	logger.Tracef(ctx, "processingLoop")
	defer func() { logger.Tracef(ctx, "/processingLoop") }()

	t := time.NewTicker(IterationInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			err := stt.commitAudio(ctx)
			if err != nil {
				logger.Debugf(ctx, "unable to commit audio: %v", err)
				stt.Mutex.Do(xsync.WithNoLogging(ctx, true), func() {
					stt.CommitAudioError = err
				})
				return
			}
		}
	}
}

func (stt *SpeechToText) Close() error {
	stt.CancelFunc()
	return nil
}

func (*SpeechToText) AudioEncoding(context.Context) (audio.Encoding, error) {
	return (*SpeechToText)(nil).AudioEncodingNoErr(), nil
}

func (*SpeechToText) AudioChannels(context.Context) (audio.Channel, error) {
	return (*SpeechToText)(nil).AudioChannelsNoErr(), nil
}

func (*SpeechToText) AudioEncodingNoErr() audio.EncodingPCM {
	return audio.EncodingPCM{
		PCMFormat:  audio.PCMFormatFloat32LE,
		SampleRate: 16000,
	}
}

func (*SpeechToText) AudioChannelsNoErr() audio.Channel {
	return 1
}

func (stt *SpeechToText) writeSegment(
	ctx context.Context,
	s *whisper.Segment,
	isFinal bool,
) bool {
	return xsync.DoA3R1(xsync.WithNoLogging(ctx, true), &stt.Mutex, stt.writeSegmentNoLock, ctx, s, isFinal)
}

func containsAlphaNum(s string) bool {
	return strings.ContainsFunc(s, func(r rune) bool {
		if r == '-' {
			return false
		}
		return unicode.IsLetter(r) || unicode.IsDigit(r)
	})
}

func (stt *SpeechToText) writeSegmentNoLock(
	ctx context.Context,
	s *whisper.Segment,
	isFinal bool,
) bool {
	logger.Debugf(ctx, "segment: %#+v; isFinal: %v", s, isFinal)

	trimmedText := strings.ToLower(strings.Trim(s.Text, " "))
	switch {
	case strings.HasPrefix(trimmedText, "[") && strings.HasSuffix(trimmedText, "]"):
		// e.g.: [silence], [typing], [click], [music], [blank_audio], [ pause ]
		return false
	case strings.HasPrefix(trimmedText, "(") && strings.HasSuffix(trimmedText, ")"):
		// e.g.: (clicking), (faint clicking), (door opens)
		return false
	case strings.HasPrefix(trimmedText, "*") && strings.HasSuffix(trimmedText, "*"):
		// e.g.: *thump*
		return false
	case strings.HasPrefix(trimmedText, "♪") && strings.HasSuffix(trimmedText, "♪"):
		// e.g.: ♪ ♪
		return false
	}

	if s.SpeakerTurn {
		stt.IsFirstSpeakerSpeaking = !stt.IsFirstSpeakerSpeaking
	}

	speaker := ">"
	if !stt.IsFirstSpeakerSpeaking {
		speaker = "<"
	}

	nonEmptyTokenCount := 0

	committingPos := getDurationFromBytes(stt.CommittingPosBytes)
	words := make([]speech.TranscriptToken, 0, len(s.Tokens))
	for idx, token := range s.Tokens {
		logger.Debugf(ctx, "token %d: %#+v", idx, token)
		words = append(words, speech.TranscriptToken{
			StartTime:  token.T0 + committingPos,
			EndTime:    token.T1 + committingPos,
			Text:       speech.Text(token.Text),
			Confidence: token.P,
			Speaker:    speaker,
		})
		if containsAlphaNum(token.Text) {
			nonEmptyTokenCount++
		}
	}

	if nonEmptyTokenCount == 0 {
		return false
	}

	t := &speech.Transcript{
		Variants: []speech.TranscriptVariant{{
			Text:             speech.Text(s.Text),
			TranscriptTokens: words,
			Confidence:       0.5,
		}},
		Stability:           0,
		NoSpeechProbability: s.NoSpeechProb,
		AudioChannelNum:     stt.AudioChannelsNoErr(),
		Language:            speech.Language(whisper.Whisper_lang_str(stt.Context.DefaultLangId())),
		IsFinal:             isFinal,
	}

	logger.Debugf(ctx, "sending Transcript: %#+v", *t)
	select {
	case stt.Out <- t:
	default:
		logger.Error(ctx, "the queue is full, dropping the message")
	}
	return true
}

func (stt *SpeechToText) WriteAudio(
	ctx context.Context,
	frame []byte,
) (_err error) {
	logger.Tracef(ctx, "WriteAudio(ctx, frame[len:%d])", len(frame))
	return xsync.DoR1(xsync.WithNoLogging(ctx, true), &stt.Mutex, func() error {
		defer func() {
			logger.Tracef(ctx, "/WriteAudio(ctx, frame[len:%d]): %v; resulting buf len: %d (%v)", len(frame), _err, len(stt.NextBuffer), getDurationFromBytes(uint64(len(stt.NextBuffer))))
		}()
		if stt.CommitAudioError != nil {
			return fmt.Errorf("audio commit error: %w", stt.CommitAudioError)
		}
		stt.NextBuffer = append(stt.NextBuffer, frame...)

		// the buffer is already too big, assuming it is not committing, because it contains
		// essentially silence, so just cutting the buffer in half
		limit := getBytesPos(BufferLimit)
		if uint64(len(stt.NextBuffer)) > limit {
			copy(stt.NextBuffer, stt.NextBuffer[limit/2:])
			stt.NextBuffer = stt.NextBuffer[:limit/2]

			stt.CommittingPosBytes += limit / 2
			logger.Debugf(ctx, "cutting the buffer in half (newPos: %v)", stt.CommittingPosBytes)
		}

		return nil
	})
}

func (stt *SpeechToText) checkIfVoiceActive(
	ctx context.Context,
	buf []byte,
) (_ret0 bool, _ret1 time.Duration, _err error) {
	logger.Tracef(ctx, "checkIfVoiceActive")
	defer func() { logger.Tracef(ctx, "/checkIfVoiceActive: %v %v %v", _ret0, _ret1, _err) }()

	vadChannels, err := stt.VAD.Channels(ctx)
	if err != nil {
		return true, 0, fmt.Errorf("unable to get the VAD's amount of channels: %w", err)
	}

	vadEncoding, err := stt.VAD.Encoding(ctx)
	if err != nil {
		return true, 0, fmt.Errorf("unable to get the VAD's encoding: %w", err)
	}

	vadEncodingPCM, ok := vadEncoding.(audio.EncodingPCM)
	if !ok {
		return true, 0, fmt.Errorf("VAD expects a non-PCM format: %T", vadEncoding)
	}

	sttEncoding := stt.AudioEncodingNoErr()

	resampler, err := resampler.NewResampler(
		resampler.Format{
			Channels:   stt.AudioChannelsNoErr(),
			SampleRate: sttEncoding.SampleRate,
			PCMFormat:  sttEncoding.PCMFormat,
		},
		bytes.NewReader(buf),
		resampler.Format{
			Channels:   vadChannels,
			SampleRate: vadEncodingPCM.SampleRate,
			PCMFormat:  vadEncodingPCM.PCMFormat,
		},
	)
	if err != nil {
		return true, 0, fmt.Errorf("unable to initialize a resampler: %w", err)
	}

	vadSampleSize := vadEncoding.BytesPerSample() * uint(vadChannels)
	sttSampleSize := sttEncoding.BytesPerSample() * uint(stt.AudioChannelsNoErr())
	vadBufferLength := 1 + 2*float64(len(buf))*float64(vadEncodingPCM.SampleRate)/float64(sttEncoding.SampleRate)*float64(vadSampleSize)/float64(sttSampleSize) // "2*" just to make sure no rounding error causes any problems; the same go for '1+"

	if cap(stt.VADBuffer) < int(vadBufferLength) {
		stt.VADBuffer = make([]byte, int(vadBufferLength))
	} else {
		stt.VADBuffer = stt.VADBuffer[:int(vadBufferLength)]
	}

	n, err := resampler.Read(stt.VADBuffer)
	if err != nil {
		return true, 0, fmt.Errorf("unable to resample: %w", err)
	}

	if n >= len(stt.VADBuffer) {
		return true, 0, fmt.Errorf("internal error: buffer length is calculated incorrectly: %d >= %d", n, len(stt.VADBuffer))
	}

	msg := stt.VADBuffer[:n]

	maxConfidence, foundAt, err := stt.VAD.FindNextVoice(ctx, msg, stt.VADThreshold, VADMinVoiceDuration)
	if err != nil {
		return true, 0, fmt.Errorf("unable to detect voice probability: %w", err)
	}

	logger.Debugf(ctx, "VAD result: %f, thus %f?>?%f:%v", maxConfidence, maxConfidence, stt.VADThreshold, maxConfidence > stt.VADThreshold)
	return maxConfidence > stt.VADThreshold, foundAt, nil
}

func (stt *SpeechToText) commitAudio(
	ctx context.Context,
) (_err error) {
	logger.Tracef(ctx, "commitAudio")
	defer func() { logger.Tracef(ctx, "/commitAudio: %v", _err) }()

	buf := xsync.DoR1(xsync.WithNoLogging(ctx, true), &stt.Mutex, func() []byte {
		if uint64(len(stt.NextBuffer)) < requiredSendingFrameSize() {
			logger.Tracef(ctx, "buffer is not big enough: %d < %d", len(stt.NextBuffer), requiredSendingFrameSize())
			return nil
		}

		stt.NextBuffer, stt.CommittingBuffer = stt.CommittingBuffer, stt.NextBuffer
		stt.NextBuffer = stt.NextBuffer[:0]
		return stt.CommittingBuffer
	})
	if buf == nil {
		return nil
	}

	committingBuf := buf
	oldCommittedPosBytes := stt.CommittingPosBytes
	defer stt.Mutex.Do(xsync.WithNoLogging(ctx, true), func() {
		bytesDiff := stt.CommittingPosBytes - oldCommittedPosBytes
		stt.TempBuffer = stt.TempBuffer[:0]
		stt.TempBuffer = append(stt.TempBuffer, committingBuf[bytesDiff:]...)
		stt.TempBuffer = append(stt.TempBuffer, stt.NextBuffer...)
		stt.NextBuffer = stt.NextBuffer[:0]
		stt.CommittingBuffer = stt.CommittingBuffer[:0]
		stt.NextBuffer, stt.TempBuffer = stt.TempBuffer, stt.NextBuffer

		assert(ctx, len(stt.NextBuffer)%4 == 0)

		tsDiff := getDurationFromBytes(stt.CommittingPosBytes - oldCommittedPosBytes)
		logger.Debugf(
			ctx,
			"considering final everything until %v (%v); leftover buffer: %d bytes (%v)",
			tsDiff,
			getDurationFromBytes(stt.CommittingPosBytes)+tsDiff,
			len(stt.NextBuffer),
			getDurationFromBytes(uint64(len(stt.NextBuffer))),
		)
	})

	bufferEndTSDiff := getDurationFromBytes(uint64(len(buf)))

	preserveBytes := getBytesPos(PreserveHeadingDuration)

	discardBuffer := func() {
		stt.NoUsefulSegmentsIterations = 0
		stt.CommittingPosBytes += uint64(len(buf)) - preserveBytes
	}

	committingPos := getDurationFromBytes(stt.CommittingPosBytes)
	logger.Debugf(
		ctx,
		"stt.VADThreshold > 0 && stt.CommittingPos > stt.VADCacheAlreadyHasVoice: %v > 0 && %v>%v: %v",
		stt.VADThreshold, committingPos, stt.VADCacheVoiceFoundAt,
		stt.VADThreshold > 0 && committingPos > stt.VADCacheVoiceFoundAt,
	)
	if stt.VADThreshold > 0 && getDurationFromBytes(stt.CommittingPosBytes) > stt.VADCacheVoiceFoundAt {
		voiceIsActive, foundAt, err := stt.checkIfVoiceActive(
			belt.WithField(ctx, "subsystem", "VAD"),
			buf,
		)
		if err != nil {
			logger.Errorf(ctx, "VAD: unable to detect if voice is active: %v", err)
		}
		if !voiceIsActive {
			discardBuffer()
			return nil
		}
		stt.VADCacheVoiceFoundAt = getDurationFromBytes(stt.CommittingPosBytes) + foundAt

		shouldCutAway := stt.VADCacheVoiceFoundAt - PreserveHeadingDuration - committingPos
		logger.Debugf(ctx, "VAD: shouldCutAway == %v", shouldCutAway)
		if shouldCutAway > 0 {
			logger.Debugf(ctx, "VAD: cutting away %s from the beginning (%s -> %s)", shouldCutAway, bufferEndTSDiff, bufferEndTSDiff-shouldCutAway)
			bytesDiff := getBytesPos(shouldCutAway)
			stt.CommittingPosBytes += uint64(bytesDiff)
			if int(bytesDiff) >= len(buf) {
				logger.Errorf(ctx, "VAD: we removed the whole buffer; this was supposed to be impossible (we should preserve at least PreserveHeadingDuration): %d >= %d", int(bytesDiff), len(buf))
				stt.NoUsefulSegmentsIterations = 0
				return nil
			}
			buf = buf[bytesDiff:]
		}
	}

	samples := convertBytesToFloat32Slice(buf)
	defer func() {
		// nothing is supposed to change `buf` length after `samples` is already created
		assert(ctx, len(buf)/4 == len(samples), len(buf), len(buf)/4, len(samples))
	}()

	duration := getDurationFromBytes(uint64(len(buf)))
	logger.Debugf(
		ctx,
		"writing to whisper %d bytes, %v, %d samples...",
		len(buf),
		duration,
		len(samples),
	)
	stt.Iterations++
	startCommittingTS := time.Now()
	stt.Params.SetOffsetMS(0)
	err := whisper.Whisper_full(
		stt.Context,
		stt.Params,
		samples,
	)
	commitTime := time.Since(startCommittingTS)
	logger.Debugf(
		ctx,
		"finished writing to whisper %d bytes, %v, %d samples: %v (it took %v)",
		len(buf),
		duration,
		len(samples),
		err,
		commitTime,
	)
	if err != nil {
		return err
	}

	numSegments := stt.Context.NumSegments()
	logger.Debugf(ctx, "numSegments == %d", numSegments)
	if numSegments == 0 {
		discardBuffer()
		return nil
	}

	lastSegment := stt.Context.Segment(numSegments - 1)
	lastSegmentStartTS := getFirstTimestamp(lastSegment)

	var lastSegmentEndTS time.Duration
	if lastSegmentStartTS-time.Millisecond*500 < stt.LastSegmentStartTS &&
		len(lastSegment.Text) <= len(stt.LastSegmentString) {
		// for some reason on large-v3 it treats an silent tail as a part of the sentence. Forcefully cutting it here:
		lastSegmentEndTS = stt.LastSegmentEndTS
	} else {
		lastSegmentEndTS = getLastTimestamp(lastSegment)

		stt.LastSegmentString = lastSegment.Text
		stt.LastSegmentStartTS = lastSegmentStartTS
		stt.LastSegmentEndTS = lastSegmentEndTS
	}

	lastCommittingSegmentIdx := numSegments - 2
	tailGapLength := bufferEndTSDiff - lastSegmentEndTS
	logger.Debugf(ctx, "tailGapLength == %v == %v - %v", tailGapLength, bufferEndTSDiff, lastSegmentEndTS)
	if tailGapLength >= GapToCommit {
		logger.Debugf(ctx, "considering the last segment committed")
		lastCommittingSegmentIdx = numSegments - 1
	} else {
		logger.Debugf(ctx, "considering the last segment uncommitted, yet")
	}

	logger.Debugf(ctx, "lastCommittingSegmentIdx == %d", lastCommittingSegmentIdx)

	hasHangingSegment := false
	numUsefulSegments := 0
	for i := 0; i < numSegments; i++ {
		logger.Debugf(ctx, "writeSegment(ctx, stt.Context.Segment(%d), %v)", i, i <= lastCommittingSegmentIdx)
		segment := stt.Context.Segment(i)
		if isHangingSegment(segment) {
			logger.Debugf(ctx, "this is a hang-causing segment")
			if i > lastCommittingSegmentIdx {
				logger.Debugf(ctx, "setting lastCommittingSegmentIdx to %d", i)
				lastCommittingSegmentIdx = i
			}
			hasHangingSegment = true
			continue
		}
		if stt.isLikelyHallucination(ctx, segment) {
			logger.Debugf(ctx, "likely a hallucination: '%s', skipping", segment.Text)
			continue
		}
		if stt.writeSegment(ctx, segment, i <= lastCommittingSegmentIdx) {
			numUsefulSegments++
		}
	}

	logger.Debugf(ctx, "numUsefulSegments == %d", numUsefulSegments)
	if numUsefulSegments == 0 {
		stt.NoUsefulSegmentsIterations++
		logger.Debugf(
			ctx,
			"%d: NoUsefulSegmentsIterations: %d >= %d; hasHangingSegment: %v",
			stt.Iterations,
			stt.NoUsefulSegmentsIterations, DiscardIfNoUsefulSegmentsIterations,
			hasHangingSegment,
		)
		if stt.NoUsefulSegmentsIterations >= DiscardIfNoUsefulSegmentsIterations || hasHangingSegment {
			discardBuffer()
			return nil
		}
	}

	logger.Debugf(ctx, "stt.Iterations == %d", stt.Iterations)
	if stt.Iterations <= 2 { // warmup
		discardBuffer()
		return nil
	}

	var (
		tsDiff    time.Duration
		bytesDiff uint64
	)

	logger.Debugf(ctx, "resulting lastCommittingSegmentIdx == %d", lastCommittingSegmentIdx)
	if lastCommittingSegmentIdx >= 0 {
		lastCommittingSegment := stt.Context.Segment(lastCommittingSegmentIdx)
		tsDiff = getLastTimestamp(lastCommittingSegment)
		bytesDiff = getBytesPos(tsDiff)
		logger.Debugf(ctx, "lastCommittingSegment == %#+v; tsDiff == %s", lastCommittingSegment, tsDiff)
	}

	logger.Debugf(ctx, "committed up to %s (%d)", tsDiff, bytesDiff)
	assert(ctx, int(bytesDiff) < len(buf), int(bytesDiff), len(buf))
	assert(ctx, bytesDiff%4 == 0, bytesDiff)
	stt.CommittingPosBytes += bytesDiff

	return nil
}

func getDurationFromBytes(bytes uint64) time.Duration {
	stt := (*SpeechToText)(nil)
	return time.Duration(float64(time.Second) * float64(bytes) / float64(stt.AudioEncodingNoErr().BytesForSecond()))
}

func getBytesPos(d time.Duration) uint64 {
	stt := (*SpeechToText)(nil)
	return stt.AudioEncodingNoErr().BytesForDuration(d) * uint64(stt.AudioChannelsNoErr())
}

func requiredSendingFrameSize() uint64 {
	return getBytesPos(2 * time.Second)
}

func (stt *SpeechToText) OutputChan(context.Context) (<-chan *speech.Transcript, error) {
	return stt.OutputChanNoErr(), nil
}

func (stt *SpeechToText) OutputChanNoErr() <-chan *speech.Transcript {
	return stt.Out
}

func getLastTimestamp(s *whisper.Segment) time.Duration {
	for idx := len(s.Tokens) - 1; idx >= 0; idx-- {
		token := s.Tokens[idx]
		if token.T0 == token.T1 {
			continue
		}
		return token.T1
	}
	return 0
}

func getFirstTimestamp(s *whisper.Segment) time.Duration {
	for _, token := range s.Tokens {
		if token.T0 == token.T1 {
			continue
		}
		return token.T0
	}
	return 0
}
