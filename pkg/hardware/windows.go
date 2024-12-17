//go:build windows
// +build windows

package hardware

import (
	"github.com/yusufpapurcu/wmi"
)

type windows struct {
}

func init() {
	resister("windows", &windows{})
}

type Win32_Processor struct {
	Name string
}

type Win32_VideoController struct {
	Name string
}

func (*windows) GPUModel() string {

	var videoControllers []Win32_VideoController
	query := wmi.CreateQuery(&videoControllers, "")
	err := wmi.Query(query, &videoControllers)
	if err != nil {
		return ""
	}
	if len(videoControllers) > 0 {
		return videoControllers[0].Name
	} else {
		return ""
	}

}
func (*windows) CPUModel() string {

	var processors []Win32_Processor
	query := wmi.CreateQuery(&processors, "")
	err := wmi.Query(query, &processors)
	if err != nil {
		return ""
	}
	if len(processors) > 0 {
		return processors[0].Name
	} else {
		return ""
	}

}
