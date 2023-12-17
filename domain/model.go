package domain

type SegmentType string

const (
	AudioSegmentType SegmentType = "audio"
	ImageSegmentType SegmentType = "image"
)

type SegmentWithMedia struct {
	MediaContent []byte
	Segment
}

func NewSegment(text string, segmentType SegmentType, id string, storyID string, ordinal int) Segment {
	return Segment{
		Text:    text,
		Type:    segmentType,
		ID:      id,
		StoryID: storyID,
		Ordinal: ordinal,
	}
}

type Segment struct {
	Text    string
	Type    SegmentType
	ID      string
	StoryID string
	Ordinal int
}

type SegmentEvent struct {
	StoryId   string      `json:"story_id"`
	SegmentId string      `json:"segment_id"`
	Text      string      `json:"text"`
	Type      SegmentType `json:"type"`
	Ordinal   int         `json:"ordinal"`
	Url       string      `json:"url"`
}

type EndGenerationEvent struct {
	MessageEvent
}

type ErrorEvent struct {
	MessageEvent
}

type MessageEvent struct {
	StoryID string `json:"story_id"`
	Message string `json:"message"`
}

type SegmentWithMediaUrl struct {
	MediaURL string
	Segment
}

func (s SegmentWithMediaUrl) ToEvent() SegmentEvent {
	return SegmentEvent{
		StoryId:   s.StoryID,
		SegmentId: s.ID,
		Text:      s.Text,
		Type:      s.Type,
		Ordinal:   s.Ordinal,
		Url:       s.MediaURL,
	}
}
