package engine

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/execlens/execlens/internal/database"
)

// CollectSystemInfo inspects the host machine for environmental metadata.
func CollectSystemInfo() database.SystemInfo {
	info := database.SystemInfo{
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		LogicalCPUs: runtime.NumCPU(),
		CPUModel:    getCPUModel(),
		TotalMemory: getTotalMemoryBytes(),
	}

	if hostname, err := os.Hostname(); err == nil {
		info.Hostname = hostname
	}

	return info
}

func getCPUModel() string {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return runtime.GOARCH
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return runtime.GOARCH
}

func getTotalMemoryBytes() uint64 {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if kb, err := strconv.ParseUint(parts[1], 10, 64); err == nil {
					return kb * 1024
				}
			}
		}
	}
	return 0
}
