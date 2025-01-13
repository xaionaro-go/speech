package whisper

import (
	"context"

	"github.com/facebookincubator/go-belt/tool/logger"
)

func assert(
	ctx context.Context,
	shouldBeTrue bool,
	args ...any,
) {
	if shouldBeTrue {
		return
	}

	if len(args) > 0 {
		logger.Panicf(ctx, "assertion failed: %v", args)
	}
	logger.Panicf(ctx, "assertion failed")
}
