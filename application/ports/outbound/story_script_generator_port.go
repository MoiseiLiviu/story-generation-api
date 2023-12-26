package outbound

import "context"

type GenerateStoryScriptRequest struct {
	Input         string
	WordsPerStory int
}

type StoryScriptGeneratorPort interface {
	Generate(ctx context.Context, req GenerateStoryScriptRequest) (<-chan string, <-chan error)
}
