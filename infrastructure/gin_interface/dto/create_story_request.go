package dto

type CreateStoryRequest struct {
	Input   string `json:"input" binding:"required"`
	VoiceID string `json:"voice_id" binding:"required"`
}
