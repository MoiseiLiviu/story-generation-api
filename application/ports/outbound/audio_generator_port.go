package outbound

import (
	"context"
	"io"
)

type GenerateAudioRequest struct {
	Text    string
	VoiceID string
}

type AudioGeneratorPort interface {
	Generate(ctx context.Context, req GenerateAudioRequest) (io.ReadCloser, error)
}
