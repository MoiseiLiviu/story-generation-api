package main

type Segment struct {
	StoryId   string `json:"story_id"`
	SegmentId string `json:"segment_id"`
	Text      string `json:"text"`
	Type      string `json:"type"`
	Ordinal   int    `json:"ordinal"`
	Url       string `json:"url"`
}

type MockSegment struct {
	Segment
	Delay int `json:"delay"`
}
