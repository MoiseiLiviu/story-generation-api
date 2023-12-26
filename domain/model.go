package domain

type SegmentType string

const (
	AudioSegmentType SegmentType = "audio"
	ImageSegmentType SegmentType = "image"
)

type VideoSegment struct {
	FileName string
	Ordinal  int
	Text     string
	Duration float64
	ID       string
}

type VideoSegmentsAscByOrdinal []VideoSegment

func (a VideoSegmentsAscByOrdinal) Len() int           { return len(a) }
func (a VideoSegmentsAscByOrdinal) Less(i, j int) bool { return a[i].Ordinal < a[j].Ordinal }
func (a VideoSegmentsAscByOrdinal) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

type AudioWithImageBackground struct {
	SegmentWithMediaFile
	BackgroundImageFileName string
}

type SegmentWithMediaFile struct {
	FileName string
	Segment
}

func (s SegmentWithMediaFile) Equals(other SegmentWithMediaFile) bool {
	return s.ID == other.ID
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

const DefaultImageID = "default"

type Segment struct {
	Text              string
	Type              SegmentType
	ID                string
	StoryID           string
	Ordinal           int
	BackgroundImageID string
}
