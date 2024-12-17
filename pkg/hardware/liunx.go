package hardware

import (
	"os/exec"
	"strings"
)

type linux struct {
}

func init() {
	resister("linux", &linux{})
}
func (*linux) GPUModel() string {
	gpuModelCmd := exec.Command("nvidia-smi", "--query-gpu=name", "--format=csv,noheader")
	gpuModelOutput, err := gpuModelCmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(gpuModelOutput))
}
func (*linux) CPUModel() string {
	cpuModelCmd := exec.Command("cat", "/proc/cpuinfo")
	cpuModelOutput, err := cpuModelCmd.Output()
	if err != nil {
		return ""
	}
	return parseCPUInfo(string(cpuModelOutput))
}
func parseCPUInfo(cpuInfo string) string {
	lines := strings.Split(cpuInfo, "\n")
	for _, line := range lines {
		if strings.Contains(line, "model name") {
			modelSplit := strings.Split(line, ":")
			if len(modelSplit) > 1 {
				return strings.TrimSpace(modelSplit[1])
			}
		}
	}
	return ""
}
