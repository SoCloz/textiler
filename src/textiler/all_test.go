package textiler

// Those tests are based on https://github.com/ikirudennis/python-textile/blob/master/textile/tests/__init__.py

import (
	"testing"
)

func textileToHtml(input string) string {
	return string(ToHtml([]byte(input)))
}

func textileToXhtml(input string) string {
	return string(ToXhtml([]byte(input)))
}

func TestTextileHtml(t *testing.T) {
	// TODO: for now mark tests that we expect to pass explicitly
	passingTests := []int{0}
	for _, i := range passingTests {
		s := html_tests[i*2]
		actual := textileToHtml(s)
		expected := html_tests[i*2+1]
		if actual != expected {
			t.Errorf("\nExpected[%#v]\nActual  [%#v]", expected, actual)
		}
	}
}

func TestTextileXhtml(t *testing.T) {
	// TODO: for now mark tests that we expect to pass explicitly
	passingTests := []int{}
	for _, i := range passingTests {
		s := xhtml_tests[i*2]
		actual := textileToXhtml(s)
		expected := xhtml_tests[i*2+1]
		if actual != expected {
			t.Errorf("\nExpected[%#v]\nActual  [%#v]", expected, actual)
		}
	}
}
