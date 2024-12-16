package demo

import "fmt"

var index = 0

func FetchMessage() {
	index++
	fmt.Println(fmt.Sprintf("hello world %d!", index))
}
