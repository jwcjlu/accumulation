package hardware

import "runtime"

type Hardware interface {
	GPUModel() string
	CPUModel() string
}

var hardwareMap = make(map[string]Hardware)

func resister(key string, value Hardware) {
	hardwareMap[key] = value
}
func GPUModel() string {
	hardware, ok := hardwareMap[runtime.GOOS]
	if ok {
		return hardware.GPUModel()
	}
	return ""
}

func CPUModel() string {
	hardware, ok := hardwareMap[runtime.GOOS]
	if ok {
		return hardware.CPUModel()
	}
	return ""
}
