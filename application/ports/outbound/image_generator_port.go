package outbound

import "context"

type ImageGeneratorPort interface {
	Generate(ctx context.Context, description string) ([]byte, error)
}
