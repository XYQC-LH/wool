//go:build windows

package handler

import (
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type winCPUSnapshot struct {
	idle  uint64
	total uint64
}

func filetimeToUint64(ft windows.Filetime) uint64 {
	return uint64(ft.HighDateTime)<<32 + uint64(ft.LowDateTime)
}

var (
	modKernel32              = windows.NewLazySystemDLL("kernel32.dll")
	procGetSystemTimes       = modKernel32.NewProc("GetSystemTimes")
	procGlobalMemoryStatusEx = modKernel32.NewProc("GlobalMemoryStatusEx")
)

func callGetSystemTimes(idle, kernel, user *windows.Filetime) error {
	r1, _, e1 := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(idle)),
		uintptr(unsafe.Pointer(kernel)),
		uintptr(unsafe.Pointer(user)),
	)
	if r1 == 0 {
		if e1 != nil && e1 != syscall.Errno(0) {
			return e1
		}
		return syscall.EINVAL
	}
	return nil
}

func readWindowsCPUSnapshot() (*winCPUSnapshot, error) {
	var idle, kernel, user windows.Filetime
	if err := callGetSystemTimes(&idle, &kernel, &user); err != nil {
		return nil, err
	}

	idle64 := filetimeToUint64(idle)
	kernel64 := filetimeToUint64(kernel)
	user64 := filetimeToUint64(user)

	// total = kernel + user（kernel 包含 idle）
	total := kernel64 + user64
	return &winCPUSnapshot{idle: idle64, total: total}, nil
}

func getSystemCPUPercent() (float64, error) {
	s1, err := readWindowsCPUSnapshot()
	if err != nil {
		return 0, err
	}

	time.Sleep(150 * time.Millisecond)

	s2, err := readWindowsCPUSnapshot()
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

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

func callGlobalMemoryStatusEx(mem *memoryStatusEx) error {
	r1, _, e1 := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(mem)))
	if r1 == 0 {
		if e1 != nil && e1 != syscall.Errno(0) {
			return e1
		}
		return syscall.EINVAL
	}
	return nil
}

func getSystemMemoryPercent() (float64, error) {
	var mem memoryStatusEx
	mem.Length = uint32(unsafe.Sizeof(mem))
	if err := callGlobalMemoryStatusEx(&mem); err != nil {
		return 0, err
	}
	if mem.TotalPhys == 0 {
		return 0, nil
	}

	used := float64(mem.TotalPhys - mem.AvailPhys)
	total := float64(mem.TotalPhys)
	usage := used / total * 100
	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}
	return usage, nil
}
