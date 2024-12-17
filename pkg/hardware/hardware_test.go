package hardware_test

import (
	"fmt"
	"gitlab.vrviu.com/cloudgame_backend/rock-stack/pkg/hardware"
	"testing"
)

func TestHardware(t *testing.T) {
	fmt.Println(hardware.CPUModel())
	fmt.Println(hardware.GPUModel())
}
