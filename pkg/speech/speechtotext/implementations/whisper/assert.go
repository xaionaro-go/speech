package whisper

import (
	"context"

	"github.com/facebookincubator/go-belt/tool/logger"
)

func assert(
	ctx context.Context,
	shouldBeTrue bool,
) {
	if shouldBeTrue {
		return
	}

	logger.Panicf(ctx, "assertion failed")
}
