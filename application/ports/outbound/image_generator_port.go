package outbound

import (
	"context"
	"io"
)

type ImageGeneratorPort interface {
	Generate(ctx context.Context, description string) (io.ReadCloser, error)
}
