package dto

type CreateStoryRequest struct {
	Input   string `json:"input"`
	VoiceID string `json:"voice_id"`
}
