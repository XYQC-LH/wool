package model

// AudioSpeechRequest 语音合成（TTS）请求（OpenAI 风格）
// POST /v1/audio/speech
type AudioSpeechRequest struct {
	Model          string   `json:"model" binding:"required"`
	Input          string   `json:"input" binding:"required"`
	Voice          string   `json:"voice" binding:"required"`
	ResponseFormat string   `json:"response_format,omitempty"` // mp3/wav/opus 等
	Speed          *float64 `json:"speed,omitempty"`
}
