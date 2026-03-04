package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"nexus-api/internal/config"

	"github.com/redis/go-redis/v9"
)

var (
	Client *redis.Client
	ctx    = context.Background()
)

// Init 初始化 Redis 连接
func Init(cfg *config.Config) error {
	Client = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
	})

	// 测试连接
	_, err := Client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("Redis connection established successfully")
	return nil
}

// Close 关闭 Redis 连接
func Close() error {
	if Client != nil {
		return Client.Close()
	}
	return nil
}

// GetClient 获取 Redis 客户端
func GetClient() *redis.Client {
	return Client
}

// GetContext 获取上下文
func GetContext() context.Context {
	return ctx
}

// ==================== 通用操作 ====================

// Set 设置键值对
func Set(key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return Client.Set(ctx, key, data, expiration).Err()
}

// Get 获取值
func Get(key string, dest interface{}) error {
	data, err := Client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// Delete 删除键
func Delete(keys ...string) error {
	return Client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func Exists(key string) (bool, error) {
	result, err := Client.Exists(ctx, key).Result()
	return result > 0, err
}

// SetNX 设置键值对（仅当键不存在时）
func SetNX(key string, value interface{}, expiration time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, err
	}
	return Client.SetNX(ctx, key, data, expiration).Result()
}

// Incr 自增
func Incr(key string) (int64, error) {
	return Client.Incr(ctx, key).Result()
}

// IncrBy 自增指定值
func IncrBy(key string, value int64) (int64, error) {
	return Client.IncrBy(ctx, key, value).Result()
}

// Decr 自减
func Decr(key string) (int64, error) {
	return Client.Decr(ctx, key).Result()
}

// DecrBy 自减指定值
func DecrBy(key string, value int64) (int64, error) {
	return Client.DecrBy(ctx, key, value).Result()
}

// Expire 设置过期时间
func Expire(key string, expiration time.Duration) error {
	return Client.Expire(ctx, key, expiration).Err()
}

// TTL 获取剩余过期时间
func TTL(key string) (time.Duration, error) {
	return Client.TTL(ctx, key).Result()
}

// ==================== Hash 操作 ====================

// HSet 设置 Hash 字段
func HSet(key string, field string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return Client.HSet(ctx, key, field, data).Err()
}

// HGet 获取 Hash 字段
func HGet(key string, field string, dest interface{}) error {
	data, err := Client.HGet(ctx, key, field).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// HGetAll 获取所有 Hash 字段
func HGetAll(key string) (map[string]string, error) {
	return Client.HGetAll(ctx, key).Result()
}

// HDel 删除 Hash 字段
func HDel(key string, fields ...string) error {
	return Client.HDel(ctx, key, fields...).Err()
}

// HExists 检查 Hash 字段是否存在
func HExists(key string, field string) (bool, error) {
	return Client.HExists(ctx, key, field).Result()
}

// HIncrBy 自增 Hash 字段
func HIncrBy(key string, field string, value int64) (int64, error) {
	return Client.HIncrBy(ctx, key, field, value).Result()
}

// ==================== List 操作 ====================

// LPush 从左侧推入列表
func LPush(key string, values ...interface{}) error {
	return Client.LPush(ctx, key, values...).Err()
}

// RPush 从右侧推入列表
func RPush(key string, values ...interface{}) error {
	return Client.RPush(ctx, key, values...).Err()
}

// LPop 从左侧弹出
func LPop(key string) (string, error) {
	return Client.LPop(ctx, key).Result()
}

// RPop 从右侧弹出
func RPop(key string) (string, error) {
	return Client.RPop(ctx, key).Result()
}

// LRange 获取列表范围
func LRange(key string, start, stop int64) ([]string, error) {
	return Client.LRange(ctx, key, start, stop).Result()
}

// LLen 获取列表长度
func LLen(key string) (int64, error) {
	return Client.LLen(ctx, key).Result()
}

// ==================== Set 操作 ====================

// SAdd 添加集合成员
func SAdd(key string, members ...interface{}) error {
	return Client.SAdd(ctx, key, members...).Err()
}

// SMembers 获取集合所有成员
func SMembers(key string) ([]string, error) {
	return Client.SMembers(ctx, key).Result()
}

// SIsMember 检查是否是集合成员
func SIsMember(key string, member interface{}) (bool, error) {
	return Client.SIsMember(ctx, key, member).Result()
}

// SRem 移除集合成员
func SRem(key string, members ...interface{}) error {
	return Client.SRem(ctx, key, members...).Err()
}

// SCard 获取集合大小
func SCard(key string) (int64, error) {
	return Client.SCard(ctx, key).Result()
}

// ==================== Sorted Set 操作 ====================

// ZAdd 添加有序集合成员
func ZAdd(key string, members ...redis.Z) error {
	return Client.ZAdd(ctx, key, members...).Err()
}

// ZRange 获取有序集合范围
func ZRange(key string, start, stop int64) ([]string, error) {
	return Client.ZRange(ctx, key, start, stop).Result()
}

// ZRangeWithScores 获取有序集合范围（带分数）
func ZRangeWithScores(key string, start, stop int64) ([]redis.Z, error) {
	return Client.ZRangeWithScores(ctx, key, start, stop).Result()
}

// ZRem 移除有序集合成员
func ZRem(key string, members ...interface{}) error {
	return Client.ZRem(ctx, key, members...).Err()
}

// ZScore 获取成员分数
func ZScore(key string, member string) (float64, error) {
	return Client.ZScore(ctx, key, member).Result()
}

// ZCard 获取有序集合大小
func ZCard(key string) (int64, error) {
	return Client.ZCard(ctx, key).Result()
}

// ==================== 缓存键定义 ====================

const (
	// Token 相关
	KeyTokenPrefix     = "token:"      // token:{key} -> TokenInfo
	KeyTokenUserPrefix = "token:user:" // token:user:{user_id} -> []string (token keys)

	// 用户相关
	KeyUserPrefix    = "user:"      // user:{id} -> User
	KeyUserSession   = "session:"   // session:{session_id} -> UserSession
	KeyUserRateLimit = "rate:user:" // rate:user:{user_id} -> int (request count)
	KeyIPRateLimit   = "rate:ip:"   // rate:ip:{ip} -> int (request count)

	// 渠道相关
	KeyChannelPrefix   = "channel:"        // channel:{id} -> Channel
	KeyChannelList     = "channels:list"   // channels:list -> []Channel
	KeyChannelModels   = "channel:models:" // channel:models:{channel_id} -> []string
	KeyHealthyChannels = "channels:healthy"
	KeyChannelLatency  = "channel:latency:" // channel:latency:{channel_id} -> int (ms)

	// 模型相关
	KeyModelPrefix = "model:"      // model:{id} -> Model
	KeyModelList   = "models:list" // models:list -> []Model

	// 资源池相关
	KeyResourceAccountPrefix = "resource:"         // resource:{id} -> ResourceAccount
	KeyResourcePool          = "resource:pool:"    // resource:pool:{channel_id} -> []ResourceAccount
	KeyResourceSession       = "resource:session:" // resource:session:{account_id} -> SessionInfo

	// 统计相关
	KeyDailyStats   = "stats:daily:"   // stats:daily:{date} -> DailyStats
	KeyHourlyStats  = "stats:hourly:"  // stats:hourly:{date}:{hour} -> HourlyStats
	KeyModelStats   = "stats:model:"   // stats:model:{model}:{date} -> ModelStats
	KeyChannelStats = "stats:channel:" // stats:channel:{channel_id}:{date} -> ChannelStats

	// 分布式锁
	KeyLockPrefix = "lock:" // lock:{resource} -> string (holder id)
)

// ==================== 分布式锁 ====================

// Lock 获取分布式锁
func Lock(key string, value string, expiration time.Duration) (bool, error) {
	return Client.SetNX(ctx, KeyLockPrefix+key, value, expiration).Result()
}

// Unlock 释放分布式锁
func Unlock(key string, value string) error {
	// 使用 Lua 脚本确保只有持有者才能释放锁
	script := redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`)
	_, err := script.Run(ctx, Client, []string{KeyLockPrefix + key}, value).Result()
	return err
}

// ==================== 令牌桶限流 ====================

// TokenBucket 令牌桶
type TokenBucket struct {
	Key        string
	Capacity   int64         // 桶容量
	Rate       int64         // 每秒填充速率
	Expiration time.Duration // 过期时间
}

// NewTokenBucket 创建令牌桶
func NewTokenBucket(key string, capacity, rate int64, expiration time.Duration) *TokenBucket {
	return &TokenBucket{
		Key:        key,
		Capacity:   capacity,
		Rate:       rate,
		Expiration: expiration,
	}
}

// Allow 检查是否允许请求
func (tb *TokenBucket) Allow(tokens int64) (bool, error) {
	script := redis.NewScript(`
		local key = KEYS[1]
		local capacity = tonumber(ARGV[1])
		local rate = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local requested = tonumber(ARGV[4])
		local expiration = tonumber(ARGV[5])

		local bucket = redis.call("hmget", key, "tokens", "last_time")
		local current_tokens = tonumber(bucket[1])
		local last_time = tonumber(bucket[2])

		if current_tokens == nil then
			current_tokens = capacity
			last_time = now
		end

		local elapsed = now - last_time
		local new_tokens = math.min(capacity, current_tokens + elapsed * rate / 1000)

		if new_tokens >= requested then
			new_tokens = new_tokens - requested
			redis.call("hmset", key, "tokens", new_tokens, "last_time", now)
			redis.call("expire", key, expiration)
			return 1
		else
			redis.call("hmset", key, "tokens", new_tokens, "last_time", now)
			redis.call("expire", key, expiration)
			return 0
		end
	`)

	now := time.Now().UnixMilli()
	result, err := script.Run(ctx, Client, []string{tb.Key}, tb.Capacity, tb.Rate, now, tokens, int64(tb.Expiration.Seconds())).Int64()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

// ==================== 滑动窗口限流 ====================

// SlidingWindowRateLimit 滑动窗口限流
func SlidingWindowRateLimit(key string, limit int64, window time.Duration) (bool, error) {
	now := time.Now().UnixMilli()
	windowStart := now - window.Milliseconds()

	pipe := Client.Pipeline()

	// 移除窗口外的请求
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

	// 获取当前窗口内的请求数
	countCmd := pipe.ZCard(ctx, key)

	// 添加当前请求
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})

	// 设置过期时间
	pipe.Expire(ctx, key, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	count := countCmd.Val()
	return count < limit, nil
}

// FixedWindowRateLimit 固定窗口限流（支持一次性消耗多个额度）
//
// 说明：
// - 以 key 的首次请求时间作为窗口起点（TTL=window）
// - 原子检查 + 增量写入：不会把“被拒绝的请求”计入窗口
func FixedWindowRateLimit(key string, limit int64, window time.Duration, increment int64) (bool, error) {
	if limit <= 0 || window <= 0 {
		return true, nil
	}
	if increment <= 0 {
		increment = 1
	}

	script := redis.NewScript(`
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local window_ms = tonumber(ARGV[2])
		local increment = tonumber(ARGV[3])

		local current = tonumber(redis.call("get", key) or "0")
		if current + increment > limit then
			return 0
		end

		current = redis.call("incrby", key, increment)
		if current == increment then
			redis.call("pexpire", key, window_ms)
		end
		return 1
	`)

	windowMs := window.Milliseconds()
	result, err := script.Run(ctx, Client, []string{key}, limit, windowMs, increment).Int64()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}
