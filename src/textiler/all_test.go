package textiler

// Those tests are based on https://github.com/ikirudennis/python-textile/blob/master/textile/tests/__init__.py

import (
	"bytes"
	"testing"
)

func textileToHtml(input string) string {
	return string(ToHtml([]byte(input), false, false))
}

func textileToXhtml(input string) string {
	return string(ToXhtml([]byte(input), false, false))
}

func TestTextileHtml(t *testing.T) {
	// TODO: for now mark tests that we expect to pass explicitly
	passingTests := []int{0}
	for _, i := range passingTests {
		s := HtmlTests[i*2]
		actual := textileToHtml(s)
		expected := HtmlTests[i*2+1]
		if actual != expected {
			t.Errorf("\nExpected[%#v]\nActual  [%#v]", expected, actual)
		}
	}
}

func TestSer(t *testing.T) {
	data := []string{
		"__f__", "<i>f</i>",
		"____", "<i></i>",
		"____rest", "<i></i>rest",
		"before__ol__", "before<i>ol</i>",
		"foo**bold**is here", "foo<b>bold</b>is here",
	}
	for i := 0; i < len(data)/2; i++ {
		var buf bytes.Buffer
		serLine([]byte(data[i*2]), &buf)
		expected := []byte(data[i*2+1])
		actual := buf.Bytes()
		if !bytes.Equal(expected, actual) {
			t.Errorf("\nExpected[%s]\nActual  [%s]", string(expected), string(actual))
		}
	}
}

func TestItalic(t *testing.T) {
	italics := []string{
		"____", "", "",
		"__f__", "f", "",
		"__foo__o", "foo", "o",
		"__a_d___lo", "a_d", "_lo",
	}
	for i := 0; i < len(italics)/3; i++ {
		r1, r2 := isItalic([]byte(italics[i*3]))
		er1, er2 := []byte(italics[i*3+1]), []byte(italics[i*3+2])
		if !bytes.Equal(r1, er1) {
			t.Errorf("\nExpected[%#v]\nActual  [%#v]", er1, r1)
		}
		if !bytes.Equal(r2, er2) {
			t.Errorf("\nExpected[%#v]\nActual  [%#v]", er2, r2)
		}
	}
}

func TestTextileXhtml(t *testing.T) {
	// TODO: for now mark tests that we expect to pass explicitly
	passingTests := []int{0}
	for _, i := range passingTests {
		s := XhtmlTests[i*2]
		actual := textileToXhtml(s)
		expected := XhtmlTests[i*2+1]
		if actual != expected {
			t.Errorf("\nExpected[%#v]\nActual  [%#v]", expected, actual)
		}
	}
}
