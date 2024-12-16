package main

import (
	"accumulation/framework/ebpf/uprobe/demo"
	"fmt"
	"time"
)

func main() {
	index := 0
	for {
		time.Sleep(20 * time.Second)
		demo.FetchMessage()
		demo.FetchMessageRet(fmt.Sprintf("hello world %d", index))
		index++
	}

}
