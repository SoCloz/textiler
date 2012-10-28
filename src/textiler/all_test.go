package textiler

// Those tests are based on https://github.com/ikirudennis/python-textile/blob/master/textile/tests/__init__.py

import (
	"bytes"
	"fmt"
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

func TestUrlRef(t *testing.T) {
	data := []string{
		"[hobix]http://hobix.com", "hobix", "http://hobix.com",
		"[]http://hobix.com", "", "http://hobix.com",
	}
	for i := 0; i < len(data)/3; i++ {
		title, url := isUrlRef([]byte(data[i*3]))
		expectedTitle := data[i*3+1]
		expectedUrl := data[i*3+2]
		if !bytes.Equal(title, []byte(expectedTitle)) {
			t.Fatalf("\nExpected[%s]\nActual  [%s]", expectedTitle, string(title))
		}
		if !bytes.Equal(url, []byte(expectedUrl)) {
			t.Fatalf("\nExpected[%s]\nActual  [%s]", expectedUrl, string(url))
		}
	}
}

func TestIsSpan(t *testing.T) {
	data := []string{
		"%{color:red}%", "color:red", "", "",
		"%{color:red}foo%", "color:red", "foo", "",
		"%{color:red}inside%after", "color:red", "inside", "after",
	}
	var expected string
	for i := 0; i < len(data)/4; i++ {
		inside, style, rest := isSpanWithStyle([]byte(data[i*4]))
		expected = data[i*4+1]
		if !bytes.Equal(style, []byte(expected)) {
			t.Fatalf("\nExpected[%s]\nActual  [%s]", expected, string(style))
		}
		expected = data[i*4+2]
		if !bytes.Equal(inside, []byte(expected)) {
			t.Fatalf("\nExpected[%s]\nActual  [%s]", expected, string(inside))
		}
		expected = data[i*4+3]
		if !bytes.Equal(rest, []byte(expected)) {
			t.Fatalf("\nExpected[%s]\nActual  [%s]", expected, string(rest))
		}
	}
}

func TestIsHLine(t *testing.T) {
	data := []string{
		"h1. foo", "1", "foo",
		"h0. bar", "", "",
		"h3.rest", "", "",
		"h6. loh", "6", "loh",
	}
	for i := 0; i < len(data)/3; i++ {
		n, rest := isHLine([]byte(data[i*3]))
		expectedN := data[i*3+1]
		if n < 0 {
			if expectedN != "" {
				t.Fatalf("\nExpected[%s]\nActual  [%d]", expectedN, n)
			}
		} else {
			if expectedN != fmt.Sprintf("%d", n) {
				t.Fatalf("\nExpected[%s]\nActual  [%d]", expectedN, n)
			}
			expectedRest := data[i*3+2]
			if !bytes.Equal(rest, []byte(expectedRest)) {
				t.Fatalf("\nExpected[%s]\nActual  [%s]", expectedRest, string(rest))
			}
		}
	}
}

func TestUrl(t *testing.T) {
	data := []string{
		`"Hobix":http://hobix.com/`, "Hobix", "http://hobix.com/", "",
		`"":http://foo end`, "", "http://foo", " end",
		`"foo":Bar tender`, "foo", "Bar", " tender",
	}
	for i := 0; i < len(data)/4; i++ {
		title, url, rest := isUrlOrRefName([]byte(data[i*4]))
		titleExpected := data[i*4+1]
		urlExpected := data[i*4+2]
		restExpected := data[i*4+3]
		if !bytes.Equal(title, []byte(titleExpected)) {
			t.Fatalf("\nExpected1[%s]\nActual   [%s]", string(titleExpected), string(title))
		}
		if !bytes.Equal(url, []byte(urlExpected)) {
			t.Fatalf("\nExpected2[%s]\nActual   [%s]", string(urlExpected), string(url))
		}
		if !bytes.Equal(rest, []byte(restExpected)) {
			t.Fatalf("\nExpected3[%s]\nActual   [%s]", string(restExpected), string(rest))
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
		p := NewParserWithRenderer(false)
		s := data[i*2]
		p.serLine([]byte(s))
		expected := []byte(data[i*2+1])
		actual := p.out.Bytes()
		if !bytes.Equal(expected, actual) {
			t.Fatalf("\nTextile[%s]\nExpected[%s]\nActual  [%s]", s, string(expected), string(actual))
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
			t.Fatalf("\nExpected[%#v]\nActual  [%#v]", er1, r1)
		}
		if !bytes.Equal(r2, er2) {
			t.Fatalf("\nExpected[%#v]\nActual  [%#v]", er2, r2)
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
			t.Fatalf("\nExpected[%#v]\nActual  [%#v]", expected, actual)
		}
	}
}
