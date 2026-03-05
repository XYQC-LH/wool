package handler

import (
	"testing"

	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestNormalizeGRPCGatewayMethod(t *testing.T) {
	require.Equal(t, model.OperationChatCompletions, normalizeGRPCGatewayMethod("chat.completions"))
	require.Equal(t, model.OperationCompletions, normalizeGRPCGatewayMethod("/gateway.v1.GatewayService/Completions"))
	require.Equal(t, model.OperationEmbeddings, normalizeGRPCGatewayMethod("embeddings.create"))
	require.Equal(t, "models.list", normalizeGRPCGatewayMethod("listmodels"))
	require.Equal(t, "", normalizeGRPCGatewayMethod("unknown"))
}

func TestValidateChatRequestForGateway(t *testing.T) {
	token := &model.Token{
		AllowedModels: pq.StringArray{"gpt-4o"},
	}

	errResp := validateChatRequestForGateway(&service.ChatCompletionRequest{
		Model:    "gpt-4o",
		Messages: []service.ChatMessage{{Role: "user", Content: "hi"}},
	}, token)
	require.Nil(t, errResp)

	errResp = validateChatRequestForGateway(&service.ChatCompletionRequest{
		Model: "gpt-4o",
	}, token)
	require.NotNil(t, errResp)
	require.Equal(t, model.OpenAIErrorTypeInvalidRequest, errResp.Error.Type)

	errResp = validateChatRequestForGateway(&service.ChatCompletionRequest{
		Model:    "gpt-4",
		Messages: []service.ChatMessage{{Role: "user", Content: "hi"}},
	}, token)
	require.NotNil(t, errResp)
	require.Equal(t, model.OpenAIErrorTypePermission, errResp.Error.Type)
}

