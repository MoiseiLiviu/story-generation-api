package outbound

import "context"

type StoryScriptGeneratorPort interface {
	Generate(ctx context.Context, input string) (<-chan string, <-chan error)
}
