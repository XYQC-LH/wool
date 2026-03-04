//go:build !windows

package handler

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type procCPUSnapshot struct {
	idle  uint64
	total uint64
}

func readProcCPUSnapshot() (*procCPUSnapshot, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("读取 /proc/stat 失败")
	}

	fields := strings.Fields(scanner.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return nil, fmt.Errorf("/proc/stat 格式异常")
	}

	var nums []uint64
	for _, v := range fields[1:] {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return nil, err
		}
		nums = append(nums, n)
	}

	var total uint64
	for _, n := range nums {
		total += n
	}

	// idle = idle + iowait（如果存在）
	idle := nums[3]
	if len(nums) > 4 {
		idle += nums[4]
	}

	return &procCPUSnapshot{idle: idle, total: total}, nil
}

func getSystemCPUPercent() (float64, error) {
	s1, err := readProcCPUSnapshot()
	if err != nil {
		return 0, err
	}

	time.Sleep(150 * time.Millisecond)

	s2, err := readProcCPUSnapshot()
	if err != nil {
		return 0, err
	}

	totalDelta := float64(s2.total - s1.total)
	idleDelta := float64(s2.idle - s1.idle)
	if totalDelta <= 0 {
		return 0, nil
	}

	usage := (totalDelta - idleDelta) / totalDelta * 100
	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}

	return usage, nil
}

func getSystemMemoryPercent() (float64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var totalKB uint64
	var availableKB uint64

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "MemTotal:":
			totalKB, _ = strconv.ParseUint(fields[1], 10, 64)
		case "MemAvailable:":
			availableKB, _ = strconv.ParseUint(fields[1], 10, 64)
		}

		if totalKB > 0 && availableKB > 0 {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}

	if totalKB == 0 {
		return 0, fmt.Errorf("/proc/meminfo 缺少 MemTotal")
	}

	usedKB := totalKB
	if availableKB > 0 && availableKB <= totalKB {
		usedKB = totalKB - availableKB
	}

	usage := float64(usedKB) / float64(totalKB) * 100
	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}

	return usage, nil
}
