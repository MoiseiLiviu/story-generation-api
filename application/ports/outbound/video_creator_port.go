package outbound

import (
	"generate-script-lambda/domain"
)

type VideoCreatorResponse struct {
	VideoFileName string
	VideoSegments []domain.VideoSegment
}

type VideoCreatorPort interface {
	Create(segments []domain.AudioWithImageBackground) (*VideoCreatorResponse, error)
}
