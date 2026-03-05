package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"nexus-api/internal/middleware"
	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
)

type grpcGatewayRequest struct {
	Method  string          `json:"method"`
	Stream  bool            `json:"stream,omitempty"`
	Payload json.RawMessage `json:"payload"`
}

type webSocketStreamWriter struct {
	header http.Header
	conn   *websocket.Conn
	mu     sync.Mutex
}

func newWebSocketStreamWriter(conn *websocket.Conn) *webSocketStreamWriter {
	return &webSocketStreamWriter{
		header: make(http.Header),
		conn:   conn,
	}
}

func (w *webSocketStreamWriter) Header() http.Header {
	return w.header
}

func (w *webSocketStreamWriter) WriteHeader(_ int) {}

func (w *webSocketStreamWriter) Write(p []byte) (int, error) {
	if w == nil || w.conn == nil {
		return 0, http.ErrHijacked
	}

	payload := strings.TrimSpace(string(p))
	if payload == "" {
		return len(p), nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	err := websocket.JSON.Send(w.conn, map[string]interface{}{
		"type": "chunk",
		"data": payload,
	})
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *webSocketStreamWriter) Flush() {}

// ChatCompletionsWebSocket WebSocket 长连接入口（客户端需先发送 ChatCompletionRequest JSON）
func (h *GatewayHandler) ChatCompletionsWebSocket(c *gin.Context) {
	token, ok := middleware.GetCurrentToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.NewOpenAIError("未授权的请求", model.OpenAIErrorTypeAuthentication, nil))
		return
	}

	websocket.Handler(func(conn *websocket.Conn) {
		defer conn.Close()
		conn.PayloadType = websocket.TextFrame

		var req service.ChatCompletionRequest
		if err := websocket.JSON.Receive(conn, &req); err != nil {
			sendWebSocketOpenAIError(conn, model.NewOpenAIError("无效的请求参数: "+err.Error(), model.OpenAIErrorTypeInvalidRequest, nil))
			return
		}

		if openAIErr := validateChatRequestForGateway(&req, token); openAIErr != nil {
			sendWebSocketOpenAIError(conn, openAIErr)
			return
		}
		applyGatewayControlHeadersToChat(c, &req)

		if req.Stream {
			writer := newWebSocketStreamWriter(conn)
			if err := h.gatewayService.HandleChatCompletionStream(&req, token, writer); err != nil {
				sendWebSocketOpenAIError(conn, model.NewOpenAIError(err.Error(), model.OpenAIErrorTypeServer, nil))
				return
			}

			_ = websocket.JSON.Send(conn, map[string]interface{}{"type": "done"})
			return
		}

		resp, err := h.gatewayService.HandleChatCompletion(&req, token)
		if err != nil {
			sendWebSocketOpenAIError(conn, model.NewOpenAIError(err.Error(), model.OpenAIErrorTypeServer, nil))
			return
		}

		_ = websocket.JSON.Send(conn, map[string]interface{}{
			"type": "response",
			"data": resp,
		})
	}).ServeHTTP(c.Writer, c.Request)
}

// GRPCGateway gRPC 网关兼容入口（method + payload 映射到现有 GatewayService）
func (h *GatewayHandler) GRPCGateway(c *gin.Context) {
	token, ok := middleware.GetCurrentToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.NewOpenAIError("未授权的请求", model.OpenAIErrorTypeAuthentication, nil))
		return
	}

	var req grpcGatewayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewOpenAIError("无效的请求参数: "+err.Error(), model.OpenAIErrorTypeInvalidRequest, nil))
		return
	}

	method := normalizeGRPCGatewayMethod(req.Method)
	if method == "" {
		c.JSON(http.StatusBadRequest, model.NewOpenAIError("不支持的 gRPC method: "+req.Method, model.OpenAIErrorTypeInvalidRequest, nil))
		return
	}

	switch method {
	case model.OperationChatCompletions:
		var chatReq service.ChatCompletionRequest
		if err := json.Unmarshal(req.Payload, &chatReq); err != nil {
			c.JSON(http.StatusBadRequest, model.NewOpenAIError("payload 解析失败: "+err.Error(), model.OpenAIErrorTypeInvalidRequest, nil))
			return
		}
		if req.Stream {
			chatReq.Stream = true
		}
		if openAIErr := validateChatRequestForGateway(&chatReq, token); openAIErr != nil {
			c.JSON(http.StatusBadRequest, openAIErr)
			return
		}
		applyGatewayControlHeadersToChat(c, &chatReq)

		if chatReq.Stream {
			c.Header("Content-Type", "text/event-stream")
			c.Header("Cache-Control", "no-cache")
			c.Header("Connection", "keep-alive")
			c.Header("Transfer-Encoding", "chunked")
			err := h.gatewayService.HandleChatCompletionStream(&chatReq, token, c.Writer)
			if err != nil && !c.Writer.Written() {
				WriteOpenAIError(c, err)
			}
			return
		}

		resp, err := h.gatewayService.HandleChatCompletion(&chatReq, token)
		if err != nil {
			WriteOpenAIError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"method": method, "data": resp})
		return

	case model.OperationCompletions:
		var completionReq service.CompletionRequest
		if err := json.Unmarshal(req.Payload, &completionReq); err != nil {
			c.JSON(http.StatusBadRequest, model.NewOpenAIError("payload 解析失败: "+err.Error(), model.OpenAIErrorTypeInvalidRequest, nil))
			return
		}
		if openAIErr := validateCompletionRequestForGateway(&completionReq, token); openAIErr != nil {
			c.JSON(http.StatusBadRequest, openAIErr)
			return
		}
		applyGatewayControlHeadersToCompletion(c, &completionReq)

		resp, err := h.gatewayService.HandleCompletion(&completionReq, token)
		if err != nil {
			WriteOpenAIError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"method": method, "data": resp})
		return

	case model.OperationEmbeddings:
		var embeddingReq service.EmbeddingRequest
		if err := json.Unmarshal(req.Payload, &embeddingReq); err != nil {
			c.JSON(http.StatusBadRequest, model.NewOpenAIError("payload 解析失败: "+err.Error(), model.OpenAIErrorTypeInvalidRequest, nil))
			return
		}
		if openAIErr := validateEmbeddingRequestForGateway(&embeddingReq, token); openAIErr != nil {
			c.JSON(http.StatusBadRequest, openAIErr)
			return
		}
		applyGatewayControlHeadersToEmbedding(c, &embeddingReq)

		resp, err := h.gatewayService.HandleEmbedding(&embeddingReq, token)
		if err != nil {
			WriteOpenAIError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"method": method, "data": resp})
		return

	case "models.list":
		resp, err := h.gatewayService.ListModels()
		if err != nil {
			WriteOpenAIError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"method": method, "data": resp})
		return

	default:
		c.JSON(http.StatusBadRequest, model.NewOpenAIError("不支持的 gRPC method: "+req.Method, model.OpenAIErrorTypeInvalidRequest, nil))
		return
	}
}

func sendWebSocketOpenAIError(conn *websocket.Conn, openAIErr *model.OpenAIError) {
	if conn == nil || openAIErr == nil {
		return
	}
	_ = websocket.JSON.Send(conn, map[string]interface{}{
		"type":  "error",
		"error": openAIErr.Error,
	})
}

func normalizeGRPCGatewayMethod(raw string) string {
	method := strings.TrimSpace(strings.ToLower(raw))
	if method == "" {
		return ""
	}

	switch method {
	case model.OperationChatCompletions, "chatcompletions", "chat.completion", "chat_completion", "/gateway.v1.gatewayservice/chatcompletions":
		return model.OperationChatCompletions
	case model.OperationCompletions, "completion", "completion.create", "completions.create", "/gateway.v1.gatewayservice/completions":
		return model.OperationCompletions
	case model.OperationEmbeddings, "embedding", "embeddings.create", "/gateway.v1.gatewayservice/embeddings":
		return model.OperationEmbeddings
	case "models.list", "listmodels", "/gateway.v1.gatewayservice/listmodels":
		return "models.list"
	default:
		return ""
	}
}

func validateChatRequestForGateway(req *service.ChatCompletionRequest, token *model.Token) *model.OpenAIError {
	if req == nil {
		return model.NewOpenAIError("请求不能为空", model.OpenAIErrorTypeInvalidRequest, nil)
	}
	if token == nil {
		return model.NewOpenAIError("未授权的请求", model.OpenAIErrorTypeAuthentication, nil)
	}
	if strings.TrimSpace(req.Model) == "" {
		return model.NewOpenAIError("缺少必需参数: model", model.OpenAIErrorTypeInvalidRequest, nil)
	}
	if !token.IsModelAllowed(req.Model) {
		return model.NewOpenAIError("当前 API Key 无权访问该模型", model.OpenAIErrorTypePermission, nil)
	}
	if len(req.Messages) == 0 {
		return model.NewOpenAIError("缺少必需参数: messages", model.OpenAIErrorTypeInvalidRequest, nil)
	}
	return nil
}

func validateCompletionRequestForGateway(req *service.CompletionRequest, token *model.Token) *model.OpenAIError {
	if req == nil {
		return model.NewOpenAIError("请求不能为空", model.OpenAIErrorTypeInvalidRequest, nil)
	}
	if token == nil {
		return model.NewOpenAIError("未授权的请求", model.OpenAIErrorTypeAuthentication, nil)
	}
	if strings.TrimSpace(req.Model) == "" {
		return model.NewOpenAIError("缺少必需参数: model", model.OpenAIErrorTypeInvalidRequest, nil)
	}
	if !token.IsModelAllowed(req.Model) {
		return model.NewOpenAIError("当前 API Key 无权访问该模型", model.OpenAIErrorTypePermission, nil)
	}
	return nil
}

func validateEmbeddingRequestForGateway(req *service.EmbeddingRequest, token *model.Token) *model.OpenAIError {
	if req == nil {
		return model.NewOpenAIError("请求不能为空", model.OpenAIErrorTypeInvalidRequest, nil)
	}
	if token == nil {
		return model.NewOpenAIError("未授权的请求", model.OpenAIErrorTypeAuthentication, nil)
	}
	if strings.TrimSpace(req.Model) == "" {
		return model.NewOpenAIError("缺少必需参数: model", model.OpenAIErrorTypeInvalidRequest, nil)
	}
	if !token.IsModelAllowed(req.Model) {
		return model.NewOpenAIError("当前 API Key 无权访问该模型", model.OpenAIErrorTypePermission, nil)
	}
	return nil
}
