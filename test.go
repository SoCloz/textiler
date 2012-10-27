package main

import (
	"fmt"
	"textiler"
)

func main() {
	s := "foo"
	res := textiler.ToHtml([]byte(s))
	fmt.Printf("%v\n", res)
}
