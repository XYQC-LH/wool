package scheduler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"nexus-api/internal/cache"
	"nexus-api/internal/model"
	"nexus-api/internal/repository"
)

// ModelAggregator 模型聚合器接口
// ⭐ 核心组件：将用户请求的模型名映射到所有可用的源头组（ProviderGroup）
// 维护模型与源头组的多对多关系
type ModelAggregator interface {
	// GetProviderGroups 获取模型的所有源头组
	GetProviderGroups(ctx context.Context, operation string, modelID string) ([]*model.ModelProvider, error)
	// ResolveModelAlias 解析模型别名
	ResolveModelAlias(ctx context.Context, alias string) (string, error)
	// GetAvailableProviderGroups 获取可用的源头组（过滤不可用的）
	GetAvailableProviderGroups(ctx context.Context, operation string, modelID string) ([]*model.ModelProvider, error)
	// GetProviderGroupCount 获取模型的源头组数量
	GetProviderGroupCount(ctx context.Context, operation string, modelID string) (int64, error)
}

// modelAggregator 模型聚合器实现
type modelAggregator struct {
	providerRepo repository.ModelProviderRepository
	aliasRepo    repository.ModelAliasRepository
	defaultAlias map[string]string
	cacheTTL     time.Duration
}

// NewModelAggregator 创建模型聚合器
func NewModelAggregator(providerRepo repository.ModelProviderRepository, aliasRepo repository.ModelAliasRepository) ModelAggregator {
	return &modelAggregator{
		providerRepo: providerRepo,
		aliasRepo:    aliasRepo,
		defaultAlias: defaultModelAliases(),
		cacheTTL:     time.Hour,
	}
}

// GetProviderGroups 获取模型的所有源头组
// ⭐ 核心方法：获取指定模型的所有源头组配置
func (ma *modelAggregator) GetProviderGroups(ctx context.Context, operation string, modelID string) ([]*model.ModelProvider, error) {
	operation = model.NormalizeOperation(operation)
	providers, err := ma.providerRepo.GetByModelID(ctx, operation, modelID)
	if err != nil {
		return nil, fmt.Errorf("获取模型 %s 的源头组失败: %w", modelID, err)
	}
	return providers, nil
}

// ResolveModelAlias 解析模型别名
// ⭐ 核心方法：将模型别名解析为实际的模型ID
// 支持功能：
// - gpt-4 -> gpt-4-turbo-preview
// - gpt-3.5 -> gpt-3.5-turbo
// - 自定义别名映射
//
// ⚠️ 当前实现限制：
// - 使用硬编码的别名映射，仅支持少量常见别名
// - 不支持动态配置，需要修改代码才能添加新别名
// - 没有持久化存储，重启后配置丢失
//
// 📋 未来改进方向：
// 1. 创建数据库表 model_aliases 存储别名映射
//    - 字段：id, alias, target_model, enabled, created_at, updated_at
//    - 支持通过管理界面动态添加/修改/删除别名
// 2. 使用 Redis 缓存别名映射，提高查询性能
//    - 缓存 key: model:alias:{alias}
//    - 缓存 TTL: 1小时
// 3. 支持从配置文件加载默认别名
//    - 配置文件路径：config/model_aliases.yaml
//    - 支持环境变量覆盖
func (ma *modelAggregator) ResolveModelAlias(ctx context.Context, alias string) (string, error) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return alias, nil
	}

	// 1) Redis 缓存命中（正向缓存）
	cacheKey := modelAliasCacheKey(alias)
	var cached string
	if err := cache.Get(cacheKey, &cached); err == nil {
		cached = strings.TrimSpace(cached)
		if cached != "" {
			return cached, nil
		}
	}

	// 2) 数据库别名（可动态维护，优先级最高）
	if ma.aliasRepo != nil {
		item, err := ma.aliasRepo.GetByAlias(ctx, alias)
		if err != nil {
			return "", err
		}
		if item != nil && strings.TrimSpace(item.TargetModel) != "" {
			_ = cache.Set(cacheKey, item.TargetModel, ma.cacheTTL)
			return item.TargetModel, nil
		}
	}

	// 3) 默认别名（兜底）
	if resolved, ok := ma.defaultAlias[alias]; ok {
		_ = cache.Set(cacheKey, resolved, ma.cacheTTL)
		return resolved, nil
	}

	return alias, nil
}

// GetAvailableProviderGroups 获取可用的源头组（过滤不可用的）
// ⭐ 核心方法：获取可用的源头组，过滤掉不可用的
// 过滤条件：
// - status = 'active'
// - circuit_state != 'open'（熔断器未打开）
// - circuit_open_until 为空或已过期
func (ma *modelAggregator) GetAvailableProviderGroups(ctx context.Context, operation string, modelID string) ([]*model.ModelProvider, error) {
	operation = model.NormalizeOperation(operation)
	providers, err := ma.providerRepo.GetAvailableProviders(ctx, operation, modelID)
	if err != nil {
		return nil, fmt.Errorf("获取模型 %s 的可用源头组失败: %w", modelID, err)
	}
	return providers, nil
}

// GetProviderGroupCount 获取模型的源头组数量
func (ma *modelAggregator) GetProviderGroupCount(ctx context.Context, operation string, modelID string) (int64, error) {
	providers, err := ma.GetProviderGroups(ctx, operation, modelID)
	if err != nil {
		return 0, err
	}
	return int64(len(providers)), nil
}

// ==================== 扩展功能 ====================

// ModelAliasConfig 模型别名配置
type ModelAliasConfig struct {
	Alias       string `json:"alias"`
	TargetModel string `json:"target_model"`
	Enabled     bool   `json:"enabled"`
}

// LoadAliasConfig 加载别名配置
// ⚠️ 当前实现限制：返回空列表
//
// 📋 未来实现计划：
// 1. 创建数据库表 model_aliases
//    ```sql
//    CREATE TABLE model_aliases (
//        id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
//        alias VARCHAR(100) NOT NULL UNIQUE,
//        target_model VARCHAR(100) NOT NULL,
//        enabled BOOLEAN DEFAULT TRUE,
//        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
//        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
//        INDEX idx_alias (alias),
//        INDEX idx_target_model (target_model)
//    );
//    ```
// 2. 创建 repository.ModelAliasRepository
//    - GetByAlias(ctx, alias) (*ModelAlias, error)
//    - GetAll(ctx) ([]*ModelAlias, error)
//    - Create(ctx, alias *ModelAlias) error
//    - Update(ctx, alias *ModelAlias) error
//    - Delete(ctx, id uint) error
// 3. 实现 Redis 缓存
//    - 缓存 key: model:alias:{alias}
//    - 缓存 TTL: 1小时
//    - 更新别名时清除缓存
func (ma *modelAggregator) LoadAliasConfig(ctx context.Context) ([]ModelAliasConfig, error) {
	configs := make([]ModelAliasConfig, 0, len(ma.defaultAlias))
	seen := make(map[string]struct{})

	// 先加载数据库（enabled），避免默认值覆盖实际配置
	if ma.aliasRepo != nil {
		items, err := ma.aliasRepo.ListEnabled(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			alias := strings.TrimSpace(item.Alias)
			if alias == "" {
				continue
			}
			seen[alias] = struct{}{}
			configs = append(configs, ModelAliasConfig{
				Alias:       item.Alias,
				TargetModel: item.TargetModel,
				Enabled:     item.Enabled,
			})
		}
	}

	for alias, target := range ma.defaultAlias {
		if _, ok := seen[alias]; ok {
			continue
		}
		configs = append(configs, ModelAliasConfig{
			Alias:       alias,
			TargetModel: target,
			Enabled:     true,
		})
	}

	return configs, nil
}

// UpdateAliasCache 更新别名缓存
// ⚠️ 当前实现限制：空操作
//
// 📋 未来实现计划：
// 1. 从数据库加载最新别名配置
// 2. 更新 Redis 缓存
//    - 使用 pipeline 批量更新
//    - 设置合理的 TTL（1小时）
// 3. 实现缓存预热
//    - 启动时加载常用别名到缓存
//    - 定期刷新缓存
func (ma *modelAggregator) UpdateAliasCache(ctx context.Context) error {
	configs, err := ma.LoadAliasConfig(ctx)
	if err != nil {
		return err
	}

	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}
		alias := strings.TrimSpace(cfg.Alias)
		target := strings.TrimSpace(cfg.TargetModel)
		if alias == "" || target == "" {
			continue
		}
		_ = cache.Set(modelAliasCacheKey(alias), target, ma.cacheTTL)
	}

	return nil
}

// GetModelAlias 获取模型的别名列表
// ⚠️ 当前实现限制：返回空列表
//
// 📋 未来实现计划：
// 1. 查询数据库获取所有指向该模型的别名
//    ```sql
//    SELECT alias FROM model_aliases WHERE target_model = ? AND enabled = TRUE
//    ```
// 2. 返回别名列表
// 3. 支持分页和排序
func (ma *modelAggregator) GetModelAlias(ctx context.Context, modelID string) ([]string, error) {
	if ma.aliasRepo == nil {
		return []string{}, nil
	}

	items, err := ma.aliasRepo.ListByTargetModel(ctx, modelID)
	if err != nil {
		return nil, err
	}

	aliases := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Alias) == "" {
			continue
		}
		aliases = append(aliases, item.Alias)
	}
	return aliases, nil
}

func modelAliasCacheKey(alias string) string {
	return "model:alias:" + strings.TrimSpace(alias)
}

func defaultModelAliases() map[string]string {
	return map[string]string{
		"gpt-4":       "gpt-4-turbo-preview",
		"gpt-3.5":     "gpt-3.5-turbo",
		"gpt-35":      "gpt-3.5-turbo",
		"gpt35":       "gpt-3.5-turbo",
		"gpt-4o":      "gpt-4o",
		"gpt-4o-mini": "gpt-4o-mini",
	}
}
