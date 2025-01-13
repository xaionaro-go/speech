package whisper

import (
	"fmt"
)

type ErrInitModel struct {
	Path string
	Err  error
}

func (e ErrInitModel) Error() string {
	return fmt.Sprintf("unable to initialize the model '%s': %v", e.Path, e.Err)
}

type ErrInitContext struct {
	Err error
}

func (e ErrInitContext) Error() string {
	return fmt.Sprintf("unable to initialize the context: %v", e.Err)
}

type ErrModelCannotTranslate struct{}

func (ErrModelCannotTranslate) Error() string {
	return "the provided model cannot translate"
}

type ErrInitVAD struct {
	Err error
}

func (e ErrInitVAD) Error() string {
	return fmt.Sprintf("unable to initialize VAD: %v", e.Err)
}
