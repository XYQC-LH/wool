package service

import (
	"fmt"
	"net/url"
	"strings"
)

func buildOpenAIURL(baseURL string, endpoint string) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", fmt.Errorf("base_url 不能为空")
	}

	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", fmt.Errorf("endpoint 不能为空")
	}
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("无效 base_url: %s", baseURL)
	}

	basePath := strings.TrimRight(u.Path, "/")
	// baseURL 可能已经包含 /v1（如 https://api.openai.com/v1），避免拼接出 /v1/v1
	v1Prefix := "/v1"
	if basePath == "/v1" || strings.HasSuffix(basePath, "/v1") {
		v1Prefix = ""
	}

	u.Path = basePath + v1Prefix + endpoint
	return u.String(), nil
}
