package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/observability"
	"nexus-api/internal/repository"
)

// ChannelService Channel 服务接口
type ChannelService interface {
	Create(req *model.CreateChannelRequest) (*model.Channel, error)
	GetByID(id uint) (*model.ChannelResponse, error)
	List(page, pageSize int, filters map[string]interface{}) ([]*model.ChannelResponse, *model.Pagination, error)
	Update(id uint, req *model.UpdateChannelRequest) (*model.ChannelResponse, error)
	Delete(id uint) error
	GetHealthyChannels() ([]*model.Channel, error)
	GetChannelsByModel(modelID string) ([]*model.Channel, error)
	SelectChannel(modelID string, strategy string) (*model.Channel, error)
	UpdateHealth(id uint, healthy bool, latency int) error
	TestChannel(id uint) (*ChannelTestResult, error)
	GetStats() (*ChannelStats, error)
}

// channelService Channel 服务实现
type channelService struct {
	channelRepo repository.ChannelRepository
	mu          sync.RWMutex
	roundRobin  map[string]int // 模型 -> 当前索引
}

// NewChannelService 创建 Channel 服务
func NewChannelService(channelRepo repository.ChannelRepository) ChannelService {
	return &channelService{
		channelRepo: channelRepo,
		roundRobin:  make(map[string]int),
	}
}

// ChannelTestResult 渠道测试结果
type ChannelTestResult struct {
	Success  bool   `json:"success"`
	Status   string `json:"status"`  // success | error
	Latency  int    `json:"latency"` // ms
	Message  string `json:"message"`
	Response string `json:"response,omitempty"`
}

// ChannelStats 渠道统计
type ChannelStats struct {
	TotalChannels     int64 `json:"total_channels"`
	HealthyChannels   int64 `json:"healthy_channels"`
	UnhealthyChannels int64 `json:"unhealthy_channels"`
}

// Create 创建渠道
func (s *channelService) Create(req *model.CreateChannelRequest) (*model.Channel, error) {
	channel := &model.Channel{
		Name:     req.Name,
		Type:     req.Type,
		BaseURL:  req.BaseURL,
		APIKey:   req.APIKey,
		Models:   req.Models,
		Weight:   req.Weight,
		Status:   model.ChannelStatusHealthy,
		Priority: req.Priority,
	}

	if req.Config != nil {
		channel.Config = model.JSON(req.Config)
	}

	if err := s.channelRepo.Create(channel); err != nil {
		return nil, err
	}

	return channel, nil
}

// GetByID 根据 ID 获取渠道
func (s *channelService) GetByID(id uint) (*model.ChannelResponse, error) {
	channel, err := s.channelRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, errors.New("渠道不存在")
	}

	return channel.ToResponse(), nil
}

// List 获取渠道列表
func (s *channelService) List(page, pageSize int, filters map[string]interface{}) ([]*model.ChannelResponse, *model.Pagination, error) {
	channels, total, err := s.channelRepo.List(page, pageSize, filters)
	if err != nil {
		return nil, nil, err
	}

	var responses []*model.ChannelResponse
	for _, channel := range channels {
		responses = append(responses, channel.ToResponse())
	}

	pagination := model.NewPagination(page, pageSize, total)
	return responses, pagination, nil
}

// Update 更新渠道
func (s *channelService) Update(id uint, req *model.UpdateChannelRequest) (*model.ChannelResponse, error) {
	channel, err := s.channelRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, errors.New("渠道不存在")
	}

	// 更新字段
	if req.Name != nil {
		channel.Name = *req.Name
	}
	if req.Type != "" {
		channel.Type = req.Type
	}
	if req.BaseURL != nil {
		channel.BaseURL = *req.BaseURL
	}
	if req.APIKey != nil {
		channel.APIKey = *req.APIKey
	}
	if req.Models != nil {
		channel.Models = req.Models
	}
	if req.Weight != nil {
		channel.Weight = *req.Weight
	}
	if req.Status != "" {
		channel.Status = req.Status
	}
	if req.Priority != nil {
		channel.Priority = *req.Priority
	}
	if req.Config != nil {
		channel.Config = model.JSON(req.Config)
	}

	if err := s.channelRepo.Update(channel); err != nil {
		return nil, err
	}

	return channel.ToResponse(), nil
}

// Delete 删除渠道
func (s *channelService) Delete(id uint) error {
	channel, err := s.channelRepo.GetByID(id)
	if err != nil {
		return err
	}
	if channel == nil {
		return errors.New("渠道不存在")
	}

	return s.channelRepo.Delete(id)
}

// GetHealthyChannels 获取健康的渠道
func (s *channelService) GetHealthyChannels() ([]*model.Channel, error) {
	return s.channelRepo.ListHealthy()
}

// GetChannelsByModel 根据模型获取支持的渠道
func (s *channelService) GetChannelsByModel(modelID string) ([]*model.Channel, error) {
	channels, err := s.channelRepo.ListHealthy()
	if err != nil {
		return nil, err
	}

	// 过滤支持该模型的渠道
	var supportedChannels []*model.Channel
	for _, channel := range channels {
		for _, m := range channel.Models {
			if m == modelID {
				supportedChannels = append(supportedChannels, channel)
				break
			}
		}
	}

	return supportedChannels, nil
}

// SelectChannel 选择渠道
func (s *channelService) SelectChannel(modelID string, strategy string) (*model.Channel, error) {
	channels, err := s.GetChannelsByModel(modelID)
	if err != nil {
		return nil, err
	}

	if len(channels) == 0 {
		return nil, errors.New("没有可用的渠道支持该模型")
	}

	switch strategy {
	case "priority":
		return s.selectByPriority(channels), nil
	case "round_robin":
		return s.selectByRoundRobin(modelID, channels), nil
	case "lowest_latency":
		return s.selectByLowestLatency(channels), nil
	case "weighted":
		return s.selectByWeight(channels), nil
	default:
		// 默认使用优先级策略
		return s.selectByPriority(channels), nil
	}
}

// selectByPriority 按优先级选择
func (s *channelService) selectByPriority(channels []*model.Channel) *model.Channel {
	if len(channels) == 0 {
		return nil
	}

	// 按优先级排序（数字越大优先级越高）
	sort.Slice(channels, func(i, j int) bool {
		return channels[i].Priority > channels[j].Priority
	})

	return channels[0]
}

// selectByRoundRobin 轮询选择
func (s *channelService) selectByRoundRobin(modelID string, channels []*model.Channel) *model.Channel {
	if len(channels) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.roundRobin[modelID]
	channel := channels[index%len(channels)]
	s.roundRobin[modelID] = (index + 1) % len(channels)

	return channel
}

// selectByLowestLatency 按最低延迟选择
func (s *channelService) selectByLowestLatency(channels []*model.Channel) *model.Channel {
	if len(channels) == 0 {
		return nil
	}

	// 按延迟排序
	sort.Slice(channels, func(i, j int) bool {
		return channels[i].Latency < channels[j].Latency
	})

	return channels[0]
}

// selectByWeight 按权重选择
func (s *channelService) selectByWeight(channels []*model.Channel) *model.Channel {
	if len(channels) == 0 {
		return nil
	}

	// 计算总权重
	totalWeight := 0
	for _, channel := range channels {
		totalWeight += channel.Weight
	}

	if totalWeight == 0 {
		return channels[0]
	}

	// 随机选择
	random := time.Now().UnixNano() % int64(totalWeight)
	currentWeight := int64(0)

	for _, channel := range channels {
		currentWeight += int64(channel.Weight)
		if random < currentWeight {
			return channel
		}
	}

	return channels[0]
}

// UpdateHealth 更新健康状态
func (s *channelService) UpdateHealth(id uint, healthy bool, latency int) error {
	var status model.ChannelStatus
	if healthy {
		status = model.ChannelStatusHealthy
		_ = s.channelRepo.ResetErrorCount(id)
	} else {
		status = model.ChannelStatusDown
		_ = s.channelRepo.IncrementErrorCount(id)
	}

	if err := s.channelRepo.UpdateStatus(id, status); err != nil {
		return err
	}

	if latency > 0 {
		_ = s.channelRepo.UpdateLatency(id, latency)
	}

	return nil
}

// TestChannel 测试渠道
func (s *channelService) TestChannel(id uint) (*ChannelTestResult, error) {
	channel, err := s.channelRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, errors.New("渠道不存在")
	}

	baseURL := strings.TrimRight(strings.TrimSpace(channel.BaseURL), "/")
	if baseURL == "" {
		return nil, errors.New("渠道 base_url 为空")
	}

	timeout := time.Duration(channel.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// 兼容 BaseURL 是否包含 /v1
	candidates := []string{
		baseURL + "/v1/models",
		baseURL + "/models",
	}
	if strings.HasSuffix(baseURL, "/v1") {
		candidates = []string{
			baseURL + "/models",
			strings.TrimSuffix(baseURL, "/v1") + "/v1/models",
			strings.TrimSuffix(baseURL, "/v1") + "/models",
		}
	}

	client := observability.NewHTTPClient(timeout)

	var lastLatency int
	var lastErr error
	var lastResp string

	for _, targetURL := range candidates {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
		if err != nil {
			cancel()
			lastErr = err
			continue
		}

		if channel.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+channel.APIKey)
		}

		start := time.Now()
		resp, err := client.Do(req)
		lastLatency = int(time.Since(start).Milliseconds())
		cancel()

		if err != nil {
			lastErr = err
			continue
		}

		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		_ = resp.Body.Close()
		lastResp = strings.TrimSpace(string(bodyBytes))

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			_ = s.channelRepo.UpdateLastTest(id, time.Now(), lastLatency)
			_ = s.UpdateHealth(id, true, lastLatency)
			return &ChannelTestResult{
				Success:  true,
				Status:   "success",
				Latency:  lastLatency,
				Message:  "测试成功",
				Response: lastResp,
			}, nil
		}

		lastErr = fmt.Errorf("上游返回错误: %d", resp.StatusCode)

		// 404 可能是路径不兼容，尝试下一个候选；401/403 等直接返回失败
		if resp.StatusCode == http.StatusNotFound {
			continue
		}
		break
	}

	_ = s.channelRepo.UpdateLastTest(id, time.Now(), lastLatency)
	_ = s.UpdateHealth(id, false, lastLatency)

	message := "测试失败"
	if lastErr != nil {
		message = fmt.Sprintf("%s：%s", message, lastErr.Error())
	}
	if lastResp != "" {
		message = fmt.Sprintf("%s（响应：%s）", message, lastResp)
	}

	return &ChannelTestResult{
		Success:  false,
		Status:   "error",
		Latency:  lastLatency,
		Message:  message,
		Response: lastResp,
	}, nil
}

// GetStats 获取统计信息
func (s *channelService) GetStats() (*ChannelStats, error) {
	channels, err := s.channelRepo.ListAll()
	if err != nil {
		return nil, err
	}

	stats := &ChannelStats{
		TotalChannels: int64(len(channels)),
	}

	for _, channel := range channels {
		if channel.Status == model.ChannelStatusHealthy {
			stats.HealthyChannels++
		} else {
			stats.UnhealthyChannels++
		}
	}

	return stats, nil
}
