package mock_generator

import "generate-script-lambda/domain"

type MockSegment struct {
	domain.SegmentEvent
	Delay int `json:"delay"`
}
