package dto

type CreateStoryResponse struct {
	StoryID     string `json:"story_id"`
	VideoKey    string `json:"video_key"`
	VideoRegion string `json:"video_region"`
}
