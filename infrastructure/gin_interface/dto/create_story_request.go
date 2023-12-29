package dto

type CreateStoryRequest struct {
	Input         string `json:"input" binding:"required"`
	VoiceID       string `json:"voice_id" binding:"required"`
	UserID        string `json:"user_id" binding:"required"`
	WordsPerStory int    `json:"words_per_story" binding:"required"`
}
