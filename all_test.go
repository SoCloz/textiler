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
		"%{color:red;foo:bar}foo%", "color:red;foo:bar", "foo", "",
		"%{color:red}inside%after", "color:red", "inside", "after",
	}
	var expected string
	for i := 0; i < len(data)/4; i++ {
		rest, inside, _, styleOpt, _ := isSpan([]byte(data[i*4]))
		expected = data[i*4+1]
		if !bytes.Equal(styleOpt, []byte(expected)) {
			t.Fatalf("\nExpected[%s]\nActual  [%s]", expected, string(styleOpt))
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
		"h3. rest", "3", "rest",
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

func TestSerLine(t *testing.T) {
	data := []string{
		"__f__", "<i>f</i>",
		"____", "<i></i>",
		"____rest", "<i></i>rest",
		"before__ol__", "before<i>ol</i>",
		"foo**bold**is here", "foo<b>bold</b>is here",
		`"Hobix":http://hobix.com/`, `<a href="http://hobix.com/">Hobix</a>`,
		`!http://hobix.com/sample.jpg!`, `<img src="http://hobix.com/sample.jpg" alt="">`,
		`!openwindow1.gif(Bunny.)!`, `<img src="openwindow1.gif" title="Bunny." alt="Bunny.">`,
		`!openwindow1.gif!:http://hobix.com/`, `<a href="http://hobix.com/" class="img"><img src="openwindow1.gif" alt=""></a>`,
		`@p@`, "<code>p</code>",
		`before@foo@`, "before<code>foo</code>",
		`bef@bar@after`, "bef<code>bar</code>after",
	}
	for i := 0; i < len(data)/2; i++ {
		p := NewParserWithRenderer(false)
		s := data[i*2]
		p.serLine([]byte(s))
		expected := []byte(data[i*2+1])
		actual := p.out.Bytes()
		if !bytes.Equal(expected, actual) {
			ToHtml([]byte(s), false, true)
			t.Fatalf("\nSrc:[%s]\nExp:[%s]\nGot:[%s]", s, string(expected), string(actual))
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

func TestTextileHtml(t *testing.T) {
	lastPassingTest := 12
	for i := 0; i <= lastPassingTest; i++ {
		s := HtmlTests[i*2]
		actual := textileToHtml(s)
		expected := HtmlTests[i*2+1]
		if actual != expected {
			ToHtml([]byte(s), false, true)
			t.Fatalf("\nSrc:%#v\n\nExp:%#v\n\nGot:%#v\n", s, expected, actual)
		}
	}
}

func TestTextileXhtml(t *testing.T) {
	// TODO: for now mark tests that we expect to pass explicitly
	// 4,5,6,7,8,9,10 - smartypants for '"'
	passingTests := []int{0, 1, 2, 3, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
		21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38}
	//fmt.Printf("%d xhtml tests\n", len(XhtmlTests) / 2)
	for _, i := range passingTests {
		s := XhtmlTests[i*2]
		actual := textileToXhtml(s)
		expected := XhtmlTests[i*2+1]
		if actual != expected {
			ToXhtml([]byte(s), false, true)
			t.Fatalf("\nSrc:%#v\n\nExp:%#v\n\nGot:%#v\n", s, expected, actual)
		}
	}
}
