
ANDROID_NDK_VERSION?=r27b
ANDROID_ABI?=arm64-v8a

ENABLE_DEBUG?=false
ENABLE_CUDA?=false
ENABLE_VULKAN?=false
ENABLE_BLAS?=false
ENABLE_CANN?=false
ENABLE_OPENVINO?=false
ENABLE_COREML?=false
JOBS?=$(shell nproc)

GOTAGS?=
WHISPER_BUILD_FLAGS?=
ifeq ($(ENABLE_CUDA), true)
	GOTAGS:=$(GOTAGS),cuda
	WHISPER_BUILD_FLAGS+=-DGGML_CUDA=1
else
	WHISPER_BUILD_FLAGS+=-DGGML_CUDA=0
endif
ifeq ($(ENABLE_VULKAN), true)
	WHISPER_BUILD_FLAGS+=-DGGML_VULKAN=1
else
	WHISPER_BUILD_FLAGS+=-DGGML_VULKAN=0
endif
ifeq ($(ENABLE_BLAS), true)
	WHISPER_BUILD_FLAGS+=-DGGML_BLAS=1
else
	WHISPER_BUILD_FLAGS+=-DGGML_BLAS=0
endif
ifeq ($(ENABLE_CANN), true)
	WHISPER_BUILD_FLAGS+=-DGGML_CANN=1
else
	WHISPER_BUILD_FLAGS+=-DGGML_CANN=0
endif
ifeq ($(ENABLE_OPENVINO), true)
	WHISPER_BUILD_FLAGS+=-DWHISPER_OPENVINO=1
else
	WHISPER_BUILD_FLAGS+=-DWHISPER_OPENVINO=0
endif
ifeq ($(ENABLE_COREML), true)
	WHISPER_BUILD_FLAGS+=-DWHISPER_COREML=1
else
	WHISPER_BUILD_FLAGS+=-DWHISPER_COREML=0
endif
ifeq ($(ENABLE_DEBUG), true)
	WHISPER_BUILD_FLAGS+=-DCMAKE_BUILD_TYPE=Debug
else
	WHISPER_BUILD_FLAGS+=-DCMAKE_BUILD_TYPE=Release
endif

GOTAGS:=$(GOTAGS:,%=%)

GOBUILD_FLAGS?=-buildvcs=true
ifneq ($(GOTAGS),)
	GOBUILD_FLAGS+=-tags=$(GOTAGS)
	FYNEBUILD_FLAGS+=--tags $(GOTAGS)
endif

GOPATH?=$(shell go env GOPATH)

GIT_COMMIT?=$(shell git rev-list -1 HEAD)
GOVERSION_GE_1_23=$(shell go run ./tools/goversion/ ge 1.23.0)
VERSION_STRING?=$(shell git rev-list -1 HEAD)
BUILD_DATE_STRING?=$(shell date +%s)

LINKER_FLAGS?=-X=github.com/xaionaro-go/buildvars.GitCommit=$(GIT_COMMIT) -X=github.com/xaionaro-go/buildvars.Version=$(VERSION_STRING) -X=github.com/xaionaro-go/buildvars.BuildDateString=$(BUILD_DATE_STRING)

LINKER_FLAGS_ANDROID?=$(LINKER_FLAGS)
LINKER_FLAGS_DARWIN?=$(LINKER_FLAGS)
LINKER_FLAGS_LINUX?=$(LINKER_FLAGS)
LINKER_FLAGS_WINDOWS?=$(LINKER_FLAGS) '-extldflags=$(WINDOWS_EXTLINKER_FLAGS)'

PKG_CONFIG_PATH:="$(PWD)"/pkg/speech/speechtotext/implementations/whisper/pkgconfig/:"$(PKG_CONFIG_PATH)"
WINDOWS_CGO_FLAGS?=

ifeq ($(GOVERSION_GE_1_23),true) # see https://github.com/wlynxg/anet/?tab=readme-ov-file#how-to-build-with-go-1230-or-later
	LINKER_FLAGS_ANDROID+=-checklinkname=0
endif

all: stt-$(shell go env GOOS)-$(shell go env GOARCH) sttd-$(shell go env GOOS)-$(shell go env GOARCH) subtitleswindow-$(shell go env GOOS)-$(shell go env GOARCH)

thirdparty/whisper.cpp/CMakeLists.txt:
	rm -rf thirdparty/whisper.cpp
	git submodule update --init

build:
	mkdir -p build

priv/android-apk.keystore:
	mkdir -p priv
	keytool -genkey -v -keystore priv/android-apk.keystore -alias streampanel -keyalg RSA -keysize 2048 -validity 36500

signer-sign-streampanel-arm64-apk: priv/android-apk.keystore
	jarsigner -verbose -sigalg SHA256withRSA -digestalg SHA256 -keystore priv/android-apk.keystore build/streampanel-arm64.apk streampanel

deps: thirdparty/whisper.cpp/CMakeLists.txt pkg/speech/speechtotext/implementations/whisper/pkgconfig/libwhisper.pc thirdparty/whisper.cpp/build/libwhisper-ready-CUDA_$(ENABLE_CUDA)-VULKAN_$(ENABLE_VULKAN)-BLAS_$(ENABLE_BLAS)-CANN_$(ENABLE_CANN)-OPENVINO_$(ENABLE_OPENVINO)-COREML_$(ENABLE_COREML)-ANDROIDABI_$(ANDROID_ABI)

pkg/speech/speechtotext/implementations/whisper/pkgconfig/libwhisper.pc:
	PKG_CONFIG_PATH= go generate ./pkg/speech/speechtotext/implementations/whisper/pkgconfig/...

thirdparty/whisper.cpp/build/libwhisper-ready-CUDA_$(ENABLE_CUDA)-VULKAN_$(ENABLE_VULKAN)-BLAS_$(ENABLE_BLAS)-CANN_$(ENABLE_CANN)-OPENVINO_$(ENABLE_OPENVINO)-COREML_$(ENABLE_COREML)-ANDROIDABI_$(ANDROID_ABI):
	mkdir -p thirdparty/whisper.cpp/build thirdparty/whisper.cpp/examples/whisper.android.java/app/src/main/jni/whisper/build
	cd thirdparty/whisper.cpp/build && cmake .. -DBUILD_SHARED_LIBS=OFF $(WHISPER_BUILD_FLAGS) && make -j $(JOBS)
	#cd thirdparty/whisper.cpp/examples/whisper.android.java/app/src/main/jni/whisper/build && cmake .. -DBUILD_SHARED_LIBS=OFF -DANDROID_ABI=$(ANDROID_ABI) $(WHISPER_BUILD_FLAGS) && make -j $(JOBS)
	rm -f thirdparty/whisper.cpp/build/libwhisper-ready*
	touch thirdparty/whisper.cpp/build/libwhisper-ready-CUDA_$(ENABLE_CUDA)-VULKAN_$(ENABLE_VULKAN)


sttd-linux-amd64: build deps
	$(eval INSTALL_DEST?=build/sttd-linux-amd64)
	PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build $(GOBUILD_FLAGS) -ldflags "$(LINKER_FLAGS_LINUX)" -o "$(INSTALL_DEST)" ./cmd/sttd
	$(eval undefine INSTALL_DEST)

stt-linux-amd64: build deps
	$(eval INSTALL_DEST?=build/stt-linux-amd64)
	PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build $(GOBUILD_FLAGS) -ldflags "$(LINKER_FLAGS_LINUX)" -o "$(INSTALL_DEST)" ./cmd/stt
	$(eval undefine INSTALL_DEST)

stt-windows-amd64: build deps
	$(eval INSTALL_DEST?=build/stt-windows-amd64.exe)
	PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) CGO_ENABLED=1 CGO_LDFLAGS="-static" CGO_CFLAGS="$(WINDOWS_CGO_FLAGS)" CC=x86_64-w64-mingw32-gcc GOOS=windows go build $(GOBUILD_FLAGS) -ldflags "$(LINKER_FLAGS_WINDOWS)" -o "$(INSTALL_DEST)" ./cmd/stt
	$(eval undefine INSTALL_DEST)

subtitleswindow-linux-amd64: build deps
	$(eval INSTALL_DEST?=build/subtitleswindow-linux-amd64)
	PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build $(GOBUILD_FLAGS) -ldflags "$(LINKER_FLAGS_LINUX)" -o "$(INSTALL_DEST)" ./cmd/subtitleswindow
	$(eval undefine INSTALL_DEST)

subtitleswindow-windows-amd64: build deps
	$(eval INSTALL_DEST?=build/subtitleswindow-windows-amd64.exe)
	PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) CGO_ENABLED=1 CGO_LDFLAGS="-static" CGO_CFLAGS="$(WINDOWS_CGO_FLAGS)" CC=x86_64-w64-mingw32-gcc GOOS=windows go build $(GOBUILD_FLAGS) -ldflags "$(LINKER_FLAGS_WINDOWS)" -o "$(INSTALL_DEST)" ./cmd/subtitleswindow
	$(eval undefine INSTALL_DEST)

example-stt: stt-$(shell go env GOOS)-$(shell go env GOARCH)
	$(eval WHISPER_MODEL?=medium)
	$(eval AUDIO_SOURCE_PATH?=)
	cd ./thirdparty/whisper.cpp && ./models/download-ggml-model.sh "$(WHISPER_MODEL)"

	( if [ "$(AUDIO_SOURCE_PATH)" = '' ]; then arecord -f FLOAT_LE -c 1 -r 16000; else cat "$(AUDIO_SOURCE_PATH)"; fi ) | ./build/stt-$(shell go env GOOS)-$(shell go env GOARCH) --translate=true --alignment-aheads-preset $(subst -,_,$(WHISPER_MODEL)) --print-timestamps thirdparty/whisper.cpp/models/ggml-"$(WHISPER_MODEL)".bin

example-subtitleswindow: subtitleswindow-$(shell go env GOOS)-$(shell go env GOARCH)
	$(eval WHISPER_MODEL?=medium)
	$(eval AUDIO_SOURCE_URL?=)
	cd ./thirdparty/whisper.cpp && ./models/download-ggml-model.sh "$(WHISPER_MODEL)"

	./build/subtitleswindow-$(shell go env GOOS)-$(shell go env GOARCH) --translate=true thirdparty/whisper.cpp/models/ggml-"$(WHISPER_MODEL)".bin $(AUDIO_SOURCE_URL)