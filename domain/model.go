package domain

type SegmentType string

const (
	AudioSegmentType SegmentType = "audio"
	ImageSegmentType SegmentType = "image"
)

type VideoSegment struct {
	Ordinal  int
	Text     string
	Duration float64
	ID       string
}

type AudioWithImageBackground struct {
	SegmentWithMediaFile
	BackgroundImageFileName string
}

type AudioSegmentsAscByOrdinal []AudioWithImageBackground

func (a AudioSegmentsAscByOrdinal) Len() int           { return len(a) }
func (a AudioSegmentsAscByOrdinal) Less(i, j int) bool { return a[i].Ordinal < a[j].Ordinal }
func (a AudioSegmentsAscByOrdinal) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

type SegmentWithMediaFile struct {
	FileName string
	Segment
}

func (s SegmentWithMediaFile) Equals(other SegmentWithMediaFile) bool {
	return s.ID == other.ID
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
