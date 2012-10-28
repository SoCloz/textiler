package textiler

import (
	"bytes"
	"fmt"
)

const (
	// renderer flags
	RENDERER_XHTML = 1 << iota
)

const (
	STYLE_FLOAT_RIGHT = 1
)

var newline = []byte{'\n'}

type UrlRef struct {
	url  []byte
	name []byte
}

type TextileRenderer struct {
	flags int
}

type TextileParser struct {
	r    TextileRenderer
	refs map[string]*UrlRef

	// TODO: this should be in TextileRenderer but for now it's ok
	out bytes.Buffer

	dumpLines      bool
	dumpParagraphs bool
}

func (r *TextileRenderer) isFlagSet(flag int) bool {
	return r.flags&flag != 0
}

func (r *TextileRenderer) isXhtml() bool {
	return r.isFlagSet(RENDERER_XHTML)
}

func NewTextileParser(renderer TextileRenderer) *TextileParser {
	return &TextileParser{
		r:    renderer,
		refs: make(map[string]*UrlRef),
	}
}

func lastByte(d []byte) byte {
	return d[len(d)-1]
}

func slice(d []byte, start, end int) []byte {
	if end > 0 {
		return d[start:end]
	}
	end = len(d) - 1 + end
	return d[start:end]
}

func extractStartTag(line []byte) ([]byte, bool) {
	if len(line) < 3 {
		return nil, false
	}
	if line[0] == '<' && lastByte(line) == '>' {
		return slice(line, 1, -2), true
	}
	return nil, false
}

func extractEndTag(line []byte) ([]byte, bool) {
	if len(line) < 4 {
		return nil, false
	}
	if line[0] == '<' && line[1] == '/' && lastByte(line) == '>' {
		return slice(line, 2, -2), true
	}
	return nil, false
}

func splitIntoLines(d []byte) [][]byte {
	// TODO: should handle CR, LF, CRLF
	return bytes.Split(d, []byte{'\n'})
}

func needsHtmlEscaping(b byte) []byte {
	switch b {

	/*	case '"':
		return []byte("&quot;")*/
	case '&':
		return []byte("&amp;")
	case '<':
		return []byte("&lt;")
	case '>':
		return []byte("&gt;")
	}
	return nil
}

func (p *TextileParser) serHtmlEscaped(d []byte) {
	for _, b := range d {
		if esc := needsHtmlEscaping(b); esc != nil {
			p.out.Write(esc)
		} else {
			p.out.WriteByte(b)
		}
	}
}

func needsEscaping(b byte) []byte {
	switch b {
	case '\'':
		return []byte("&#8217;")
	}
	return nil
}

func (p *TextileParser) serEscapedLine(l []byte) {
	for _, b := range l {
		if esc := needsEscaping(b); esc != nil {
			p.out.Write(esc)
		} else {
			p.out.WriteByte(b)
		}
	}
}

func (p *TextileParser) serHtmlEscapedLines(lines [][]byte) {
	for i, l := range lines {
		p.serHtmlEscaped(l)
		if i != len(lines)-1 {
			p.out.Write(newline)
		}
	}
}

// An html paragraph is where the first line is <$tag>, last line is </$tag>
func isHtmlParagraph(lines [][]byte) bool {
	if len(lines) < 2 {
		return false
	}
	tag, ok := extractStartTag(lines[0])
	if !ok {
		return false
	}
	tag2, ok := extractEndTag(lines[len(lines)-1])
	if !ok {
		return false
	}
	return bytes.Equal(tag, tag2)
}

func (p *TextileParser) serTagStartWithOptClass(tag string, class []byte) {
	p.out.WriteByte('<')
	p.out.WriteString(tag)
	if class == nil {
		p.out.WriteByte('>')
	} else {
		p.out.WriteString(fmt.Sprintf(` class="%s">`, string(class)))
	}
}

func (p *TextileParser) serTagStartWithOptStyle(tag string, style []byte) {
	p.out.WriteByte('<')
	p.out.WriteString(tag)
	if style == nil {
		p.out.WriteByte('>')
	} else {
		p.out.WriteString(fmt.Sprintf(` style="%s;">`, string(style)))
	}
}

func (p *TextileParser) serTagEnd(tag string) {
	p.out.WriteString("</")
	p.out.WriteString(tag)
	p.out.WriteByte('>')
}

func (p *TextileParser) serTag(before, inside, rest []byte, tag string) {
	p.out.Write(before) // TODO: escaped?
	p.serTagStartWithOptClass(tag, nil)
	p.serLine(inside)
	p.serTagEnd(tag)
	p.serLine(rest)
}

func (p *TextileParser) serTagWithClass(before, inside, class, rest []byte, tag string) {
	p.out.Write(before) // TODO: escaped?
	p.serTagStartWithOptClass(tag, class)
	p.serLine(inside)
	p.serTagEnd(tag)
	p.serLine(rest)
}

func (p *TextileParser) serTagWithStyle(before, inside, style, rest []byte, tag string) {
	p.out.Write(before) // TODO: escaped?
	p.serTagStartWithOptStyle(tag, style)
	p.serLine(inside)
	p.serTagEnd(tag)
	p.serLine(rest)
}

func (p *TextileParser) serSpanWithStyle(before, style, inside, rest []byte) {
	p.serEscapedLine(before)
	p.out.WriteString(fmt.Sprintf(`<span style="%s;">`, string(style)))
	p.serLine(inside)
	p.out.WriteString("</span>")
	p.serLine(rest)
}

func (p *TextileParser) serSpanWithLang(before, lang, inside, rest []byte) {
	p.serEscapedLine(before)
	p.out.WriteString(fmt.Sprintf(`<span lang="%s">`, string(lang)))
	p.serLine(inside)
	p.out.WriteString("</span>")
	p.serLine(rest)
}

func (p *TextileParser) serUrl(before, title, url, rest []byte) {
	p.serEscapedLine(before)
	p.out.WriteString(fmt.Sprintf(`<a href="%s">`, string(url)))
	p.serEscapedLine(title)
	p.out.WriteString("</a>")
	p.serLine(rest)
}

func (p *TextileParser) serImg(before []byte, imgSrc []byte, alt []byte, style int, url []byte, rest []byte) {
	p.serEscapedLine(before)
	if len(url) > 0 {
		p.out.WriteString(fmt.Sprintf(`<a href="%s" class="img">`, string(url)))
	}
	altStr := string(alt)
	styleStr := ""
	if style == STYLE_FLOAT_RIGHT {
		styleStr = ` style="float: right;"`
	}
	if len(alt) > 0 {
		p.out.WriteString(fmt.Sprintf(`<img src="%s"%s title="%s" alt="%s">`, string(imgSrc), styleStr, altStr, altStr))
	} else {
		p.out.WriteString(fmt.Sprintf(`<img src="%s"%s alt="">`, string(imgSrc), styleStr))
	}
	if len(url) > 0 {
		p.out.WriteString("</a>")
	}
	p.serLine(rest)
}

// !$imgSrc($altOptional)!:$urlOptional
// TODO: should return nil for alt instead of empty slice if not found?
func isImg(l []byte) ([]byte, []byte, int, []byte, []byte) {
	if len(l) < 3 {
		return nil, nil, 0, nil, nil
	}
	if l[0] != '!' {
		return nil, nil, 0, nil, nil
	}
	l = l[1:]
	style := 0
	if l[0] == '>' {
		style = STYLE_FLOAT_RIGHT
		l = l[1:]
	}
	var imgSrc, alt, url []byte
	endIdx := bytes.IndexByte(l, '(')
	if endIdx != -1 {
		imgSrc = l[:endIdx]
		l = l[endIdx+1:]
		endIdx = bytes.IndexByte(l, ')')
		if endIdx == -1 {
			return nil, nil, 0, nil, nil
		}
		alt = l[:endIdx]
		l = l[endIdx+1:]
		if len(l) < 1 || l[0] != '!' {
			return nil, nil, 0, nil, nil
		}
		l = l[1:]
	} else {
		endIdx = bytes.IndexByte(l, '!')
		if endIdx == -1 {
			return nil, nil, 0, nil, nil
		}
		imgSrc = l[:endIdx]
		l = l[endIdx+1:]
		alt = l[0:0]
	}
	if len(l) > 0 && l[0] == ':' {
		url, l = extractUrlOrRefName(l[1:])
	}
	return imgSrc, alt, style, url, l
}

// %{$style}$inside%$rest
func isSpanWithStyle(l []byte) ([]byte, []byte, []byte) {
	if len(l) < 4 {
		return nil, nil, nil
	}
	if l[0] != '%' && l[1] != '{' {
		return nil, nil, nil
	}
	l = l[2:]
	endIdx := bytes.IndexByte(l, '}')
	if endIdx == -1 {
		return nil, nil, nil
	}
	style := l[:endIdx]
	l = l[endIdx+1:]
	endIdx = bytes.IndexByte(l, '%')
	if endIdx == -1 {
		return nil, nil, nil
	}
	inside := l[:endIdx]
	rest := l[endIdx+1:]
	return inside, style, rest
}

// %[$lang]$inside%$rest
func isSpanWithLang(l []byte) ([]byte, []byte, []byte) {
	if len(l) < 4 {
		return nil, nil, nil
	}
	if l[0] != '%' && l[1] != '[' {
		return nil, nil, nil
	}
	l = l[2:]
	endIdx := bytes.IndexByte(l, ']')
	if endIdx == -1 {
		return nil, nil, nil
	}
	lang := l[:endIdx]
	l = l[endIdx+1:]
	endIdx = bytes.IndexByte(l, '%')
	if endIdx == -1 {
		return nil, nil, nil
	}
	inside := l[:endIdx]
	rest := l[endIdx+1:]
	return inside, lang, rest
}

// *{$style}$inside*$rest
func isStrongWithStyle(l []byte) ([]byte, []byte, []byte) {
	if len(l) < 4 {
		return nil, nil, nil
	}
	if l[0] != '*' || l[1] != '{' {
		return nil, nil, nil
	}
	l = l[2:]
	endIdx := bytes.IndexByte(l, '}')
	if endIdx == -1 {
		return nil, nil, nil
	}
	style := l[:endIdx]
	l = l[endIdx+1:]
	endIdx = bytes.IndexByte(l, '*')
	if endIdx == -1 {
		return nil, nil, nil
	}
	inside := l[:endIdx]
	rest := l[endIdx+1:]
	return inside, style, rest
}

// _($class)$inside_$rest
func isEmWithClass(l []byte) ([]byte, []byte, []byte) {
	if len(l) < 4 {
		return nil, nil, nil
	}
	if l[0] != '_' || l[1] != '(' {
		return nil, nil, nil
	}
	l = l[2:]
	endIdx := bytes.IndexByte(l, ')')
	if endIdx == -1 {
		return nil, nil, nil
	}
	class := l[:endIdx]
	l = l[endIdx+1:]
	endIdx = bytes.IndexByte(l, '_')
	if endIdx == -1 {
		return nil, nil, nil
	}
	inside := l[:endIdx]
	rest := l[endIdx+1:]
	return inside, class, rest
}

func is2Byte(l []byte, b byte) ([]byte, []byte) {
	if len(l) < 4 {
		return nil, nil
	}
	if l[0] != b || l[1] != b {
		return nil, nil
	}
	for i := 2; i < len(l)-1; i++ {
		if l[i] == b {
			if l[i+1] == b {
				return l[2:i], l[i+2:]
			}
		}
	}
	return nil, nil
}

func isItalic(l []byte) ([]byte, []byte) {
	return is2Byte(l, '_')
}

func isBold(l []byte) ([]byte, []byte) {
	return is2Byte(l, '*')
}

func (p *TextileParser) serLine(l []byte) {
	for i := 0; i < len(l); i++ {
		b := l[i]
		if b == '_' {
			if inside, rest := isItalic(l[i:]); inside != nil {
				p.serTag(l[:i], inside, rest, "i")
				return
			}
			if inside, class, rest := isEmWithClass(l[i:]); inside != nil {
				p.serTagWithClass(l[:i], inside, class, rest, "em")
				return
			}
		} else if b == '*' {
			if inside, rest := isBold(l[i:]); inside != nil {
				p.serTag(l[:i], inside, rest, "b")
				return
			}
			if inside, style, rest := isStrongWithStyle(l[i:]); inside != nil {
				p.serTagWithStyle(l[:i], inside, style, rest, "strong")
				return
			}
		} else if b == '%' {
			if inside, style, rest := isSpanWithStyle(l[i:]); inside != nil {
				p.serSpanWithStyle(l[:i], style, inside, rest)
				return
			}
			if inside, lang, rest := isSpanWithLang(l[i:]); inside != nil {
				p.serSpanWithLang(l[:i], lang, inside, rest)
				return
			}
		} else if b == '"' {
			if title, urlOrRefName, rest := isUrlOrRefName(l[i:]); title != nil {
				if urlRef, ok := p.refs[string(urlOrRefName)]; ok {
					p.serUrl(l[:i], title, urlRef.url, rest)
				} else {
					p.serUrl(l[:i], title, urlOrRefName, rest)
				}
				return
			}
		} else if b == '!' {
			if imgSrc, alt, style, url, rest := isImg(l[i:]); imgSrc != nil {
				p.serImg(l[:i], imgSrc, alt, style, url, rest)
				return
			}
		}
	}
	p.serEscapedLine(l)
}

func (p *TextileParser) serLines(lines [][]byte) {
	for i, l := range lines {
		p.serLine(l)
		if i != len(lines)-1 {
			// TODO: in xhtml mode, output "<br />"
			p.out.WriteString("<br>")
			p.out.Write(newline)
		}
	}
}

func (p *TextileParser) serHLine(n int, rest []byte) {
	p.out.WriteString(fmt.Sprintf("\t<h%d>", n))
	p.out.Write(rest) // TODO: escape?
	p.out.WriteString(fmt.Sprintf("</h%d>", n))
}

// h$n. $rest
func isHLine(l []byte) (int, []byte) {
	if len(l) < 4 {
		return -1, nil
	}
	if l[0] != 'h' || l[2] != '.' || l[3] != ' ' {
		return -1, nil
	}
	n := l[1] - '0'
	if n < 1 || n > 6 {
		return -1, nil
	}
	return int(n), l[4:]
}

// TODO: this is more complex
func isUrlEnd(b byte) bool {
	i := bytes.IndexByte([]byte{' ', '!'}, b)
	return i != -1
}

func detectUrl(l []byte) ([]byte, []byte) {
	i := bytes.Index(l, []byte{':', '/', '/'})
	if i == -1 {
		return nil, nil
	}
	s := string(l[:i])
	if !(s == "http" || s == "https") {
		return nil, nil
	}
	i += 3
	for i < len(l) {
		if isUrlEnd(l[i]) {
			return l[:i], l[i:]
		}
		i += 1
	}
	return l, l[0:0]
}

func extractUrlOrRefName(l []byte) ([]byte, []byte) {
	for i, c := range l {
		// TODO: hackish. Probably should test l[:i] against a list
		// of known refs
		if isUrlEnd(c) {
			return l[:i], l[i:]
		}
	}
	return l, l[0:0]
}

// "$title":$url or "$title":$refName
func isUrlOrRefName(l []byte) ([]byte, []byte, []byte) {
	if len(l) < 4 {
		return nil, nil, nil
	}
	if l[0] != '"' {
		return nil, nil, nil
	}
	l = l[1:]
	endIdx := bytes.IndexByte(l, '"')
	if endIdx == -1 {
		return nil, nil, nil
	}
	title := l[:endIdx]
	l = l[endIdx+1:]
	if len(l) < 1 || l[0] != ':' {
		return nil, nil, nil
	}
	urlOrRefName, rest := extractUrlOrRefName(l[1:])
	return title, urlOrRefName, rest
}

// [$name]$url
func isUrlRef(l []byte) ([]byte, []byte) {
	if len(l) < 4 {
		return nil, nil
	}
	if l[0] != '[' {
		return nil, nil
	}
	l = l[1:]
	endIdx := bytes.IndexByte(l, ']')
	if endIdx == -1 {
		return nil, nil
	}
	name := l[:endIdx]
	l = l[endIdx+1:]
	url, rest := detectUrl(l)
	if url == nil || len(rest) > 0 {
		return nil, nil
	}
	return name, url
}

func (p *TextileParser) serParagraph(lines [][]byte) {
	if len(lines) > 0 {
		if n, rest := isHLine(lines[0]); n != -1 {
			p.serHLine(n, rest)
			if len(lines) > 1 {
				p.serParagraph(lines[1:])
			}
			return
		}
	}
	p.out.WriteString("\t<p>")
	p.serLines(lines)
	p.out.WriteString("</p>")
}

func (p *TextileParser) serHtmlParagraph(lines [][]byte) {
	p.out.Write(lines[0])
	if isHtmlParagraph(lines[1 : len(lines)-1]) {
		p.out.Write(newline)
		p.serHtmlParagraph(lines[1 : len(lines)-1])
		p.out.Write(newline)
	} else {
		p.out.Write(newline)
		middleLines := lines[1 : len(lines)-1]
		p.serHtmlEscapedLines(middleLines)
		p.out.Write(newline)
	}
	p.out.Write(lines[len(lines)-1])
}

func (p *TextileParser) serParagraphs(paragraphs [][][]byte) {
	for i, para := range paragraphs {
		if i != 0 {
			p.out.Write(newline)
		}
		if isHtmlParagraph(para) {
			p.serHtmlParagraph(para)
		} else {
			p.serParagraph(para)
		}
		if i != len(paragraphs)-1 {
			p.out.Write(newline)
		}
	}
}

func groupIntoParagraphs(lines [][]byte) [][][]byte {
	currPara := make([][]byte, 0)
	res := make([][][]byte, 0)

	// paragraphs is a set of lines separated by an empty line
	for _, l := range lines {
		// TODO: html block can also signal a beginning of a new paragraph
		if len(l) == 0 {
			if len(currPara) > 0 {
				res = append(res, currPara)
			}
			// TODO: to be more efficient, reset the size to 0 instead of
			// re-allocating a new one
			currPara = make([][]byte, 0)
		}
		if len(l) > 0 {
			currPara = append(currPara, l)
		}
	}

	if len(currPara) > 0 {
		res = append(res, currPara)
	}
	return res
}

func dumpLines(lines [][]byte, out *bytes.Buffer) {
	for _, l := range lines {
		out.WriteString("'")
		out.Write(l)
		out.WriteString("'")
		out.Write(newline)
	}
}

func dumpParagraphs(paragraphs [][][]byte, out *bytes.Buffer) {
	for i, para := range paragraphs {
		isHtml := isHtmlParagraph(para)
		out.WriteString(fmt.Sprintf(":para %d, %d lines, html: %v\n", i, len(para), isHtml))
		dumpLines(para, out)
		out.Write(newline)
	}
}

func (p *TextileParser) lineExtractRef(line []byte) bool {
	if name, url := isUrlRef(line); name != nil {
		p.refs[string(name)] = &UrlRef{name: name, url: url}
		return true
	}
	return false
}

func (p *TextileParser) extractRefs(lines [][]byte) [][]byte {
	res := make([][]byte, 0)
	for _, l := range lines {
		if !p.lineExtractRef(l) {
			res = append(res, l)
		}
	}
	return res
}

func (p *TextileParser) firstPass(paragraphs [][][]byte) [][][]byte {
	res := make([][][]byte, 0)
	for _, para := range paragraphs {
		para = p.extractRefs(para)
		if len(para) > 0 {
			res = append(res, para)
		}
	}
	return res
}

func (p *TextileParser) toHtml(d []byte) []byte {

	lines := splitIntoLines(d)

	if p.dumpLines {
		var buf bytes.Buffer
		dumpLines(lines, &buf)
		fmt.Printf("%s", string(buf.Bytes()))
		return nil
	}

	paragraphs := groupIntoParagraphs(lines)
	if p.dumpParagraphs {
		var buf bytes.Buffer
		dumpParagraphs(paragraphs, &buf)
		fmt.Printf("%s", string(buf.Bytes()))
		return nil
	}

	paragraphs = p.firstPass(paragraphs)
	p.serParagraphs(paragraphs)
	return p.out.Bytes()
}

func NewParserWithRenderer(isXhtml bool) *TextileParser {
	r := TextileRenderer{}
	if isXhtml {
		r.flags = RENDERER_XHTML
	}
	return NewTextileParser(r)
}

func ToHtml(d []byte, dumpLines, dumpParagraphs bool) []byte {
	p := NewParserWithRenderer(false)
	p.dumpLines = dumpLines
	p.dumpParagraphs = dumpParagraphs
	return p.toHtml(d)
}

func ToXhtml(d []byte, dumpLines, dumpParagraphs bool) []byte {
	p := NewParserWithRenderer(true)
	p.dumpLines = dumpLines
	p.dumpParagraphs = dumpParagraphs
	return p.toHtml(d)
}
