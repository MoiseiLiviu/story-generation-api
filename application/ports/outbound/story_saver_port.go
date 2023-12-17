package outbound

import "context"

type SaveStoryParams struct {
	ID     string
	UserID string
	Input  string
}

type StorySaverPort interface {
	Save(ctx context.Context, params SaveStoryParams) error
}
