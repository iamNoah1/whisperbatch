package transcriber

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/mem"
)

const mbPerGB = 1024

// SelectModel picks the best Whisper model based on available VRAM or RAM,
// logs the decision, and returns the model name string.
func SelectModel() string {
	name, reason := selectModel(getAvailableRAMMB(), getAvailableVRAMMB())
	log.Printf("model: %s (%s)", name, reason)
	return name
}

func selectModel(ramMB, vramMB float64) (string, string) {
	if vramMB > 0 {
		switch {
		case vramMB >= 10*mbPerGB:
			return "large", fmt.Sprintf("VRAM %.0f MB >= 10 GB", vramMB)
		case vramMB >= 5*mbPerGB:
			return "medium", fmt.Sprintf("VRAM %.0f MB >= 5 GB", vramMB)
		case vramMB >= 2*mbPerGB:
			return "base", fmt.Sprintf("VRAM %.0f MB >= 2 GB", vramMB)
		default:
			// VRAM present but too small — fall through to RAM check.
		}
	}

	switch {
	case ramMB >= 16*mbPerGB:
		return "large", fmt.Sprintf("RAM %.0f MB >= 16 GB", ramMB)
	case ramMB >= 8*mbPerGB:
		return "medium", fmt.Sprintf("RAM %.0f MB >= 8 GB", ramMB)
	case ramMB >= 4*mbPerGB:
		return "base", fmt.Sprintf("RAM %.0f MB >= 4 GB", ramMB)
	default:
		return "tiny", fmt.Sprintf("RAM %.0f MB < 4 GB", ramMB)
	}
}

// getAvailableVRAMMB queries nvidia-smi for total free VRAM across all GPUs.
// Returns 0 if nvidia-smi is unavailable or reports no free memory.
func getAvailableVRAMMB() float64 {
	out, err := exec.Command(
		"nvidia-smi",
		"--query-gpu=memory.free",
		"--format=csv,noheader,nounits",
	).Output()
	if err != nil {
		return 0
	}

	var totalMB float64
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		v, err := strconv.ParseFloat(line, 64)
		if err != nil {
			continue
		}
		totalMB += v
	}
	return totalMB
}

// getAvailableRAMMB returns the available system RAM in MB.
// Returns 0 if the value cannot be determined.
func getAvailableRAMMB() float64 {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return 0
	}
	return float64(vmStat.Available) / (1024 * 1024)
}
