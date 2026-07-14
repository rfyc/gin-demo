package utils

import (
	"context"

	"github.com/spf13/cast"
)

func WorkCode(ctx context.Context) string {
	return cast.ToString(ctx.Value("workCode"))
}

func WorkName(ctx context.Context) string {
	return cast.ToString(ctx.Value("workName"))
}
