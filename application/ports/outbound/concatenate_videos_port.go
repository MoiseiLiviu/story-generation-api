package outbound

import (
	"generate-script-lambda/domain"
)

type ConcatenateVideosPort interface {
	Concatenate(segments []domain.VideoSegment) (string, error)
}
