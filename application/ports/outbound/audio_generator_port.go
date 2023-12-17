package outbound

import "context"

type GenerateAudioParams struct {
	Text    string
	VoiceID string
}

type AudioGeneratorPort interface {
	Generate(ctx context.Context, generateAudioParams GenerateAudioParams) ([]byte, error)
}
