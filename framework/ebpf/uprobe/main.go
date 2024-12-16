package main

import (
	"accumulation/framework/ebpf/uprobe/demo"
	"time"
)

func main() {
	for {
		time.Sleep(30 * time.Second)
		demo.FetchMessage()
	}

}
