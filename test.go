package main

import (
	"fmt"
	"textiler"
	"bytes"
)

func main() {
	passingTests := []int{0,1,2}

	for _, i := range passingTests {
		s := textiler.XhtmlTests[i*2]
		expected := []byte(textiler.XhtmlTests[i*2+1])
		res := textiler.ToHtml([]byte(s), false, false)
		if !bytes.Equal(res, expected) {
			textiler.ToHtml([]byte(s), false, true)
			fmt.Printf("**Conversion failed!**\n\n'%v'\n\n'%v'\n\n'%v'\n", s, string(expected), string(res))
			return
		}
	}
}
