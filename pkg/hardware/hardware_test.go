package hardware_test

import (
	"accumulation/pkg/hardware"

	"fmt"
	"testing"
)

func TestHardware(t *testing.T) {
	fmt.Println(hardware.CPUModel())
	fmt.Println(hardware.GPUModel())
}
