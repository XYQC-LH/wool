package model

import "strings"

const (
	OperationChatCompletions     = "chat.completions"
	OperationCompletions         = "completions"
	OperationEmbeddings          = "embeddings"
	OperationImagesGenerations   = "images.generations"
	OperationVideosGenerations   = "videos.generations"
	OperationAudioTranscriptions = "audio.transcriptions"
	OperationAudioTranslations   = "audio.translations"
	OperationAudioSpeech         = "audio.speech"
)

func NormalizeOperation(op string) string {
	op = strings.TrimSpace(op)
	if op == "" {
		return OperationChatCompletions
	}
	return op
}
