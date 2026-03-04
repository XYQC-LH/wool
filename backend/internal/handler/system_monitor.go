package handler

import (
	"context"
	"strconv"
	"strings"

	"nexus-api/internal/cache"
	"nexus-api/internal/database"
)

func getRedisConnectedClients() int {
	client := cache.GetClient()
	if client == nil {
		return 0
	}

	info, err := client.Info(context.Background(), "clients").Result()
	if err != nil {
		return 0
	}

	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "connected_clients:") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(line, "connected_clients:"))
		n, err := strconv.Atoi(value)
		if err != nil {
			return 0
		}
		return n
	}

	return 0
}

func getDBOpenConnections() int {
	db := database.GetDB()
	if db == nil {
		return 0
	}

	sqlDB, err := db.DB()
	if err != nil {
		return 0
	}

	return sqlDB.Stats().OpenConnections
}
