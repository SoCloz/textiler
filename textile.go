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

func extractStartTag(l []byte) ([]byte, bool) {
	if len(l) < 3 {
		return nil, false
	}
	if l[0] == '<' && lastByte(l) == '>' {
		idx := bytes.IndexByte(l, ' ')
		if idx != -1 {
			return l[1:idx], true
		}
		return slice(l, 1, -2), true
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

func isHtmlLine(l []byte) bool {
	if _, ok := extractStartTag(l); ok {
		return true
	}
	if _, ok := extractEndTag(l); ok {
		return true
	}
	return false
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

func extractUntil(l []byte, c byte) ([]byte, []byte) {
	idx := bytes.IndexByte(l, c)
	if idx == -1 {
		return nil, l
	}
	inside := l[:idx]
	rest := l[idx+1:]
	return inside, rest
}

// $start$inside$end$rest
// e.g. '@foo@bar'
func extractInside(l []byte, start, end byte) ([]byte, []byte) {
	if len(l) == 0 || l[0] != start {
		return nil, l
	}
	inside, rest := extractUntil(l[1:], end)
	if inside == nil {
		return nil, l
	}
	return inside, rest
}

// %{$style}$inside%$rest
func isSpanWithOptStyle(l []byte) ([]byte, []byte, []byte) {
	if len(l) < 3 {
		return nil, nil, nil
	}
	if l[0] != '%' {
		return nil, nil, nil
	}
	style, l := extractInside(l[1:], '{', '}')
	inside, rest := extractUntil(l, '%')
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
	l = l[1:]
	lang, l := extractInside(l, '[', ']')
	if lang == nil {
		return nil, nil, nil
	}
	inside, rest := extractUntil(l, '%')
	return inside, lang, rest
}

// *{$styleOpt}$inside*$rest
func isStrongWithOptStyle(l []byte) ([]byte, []byte, []byte) {
	if len(l) < 3 {
		return nil, nil, nil
	}
	if l[0] != '*' {
		return nil, nil, nil
	}
	style, l := extractInside(l[1:], '{', '}')
	inside, rest := extractUntil(l, '*')
	return inside, style, rest
}

// _($classOpt)$inside_$rest
func isEmWithOptClass(l []byte) ([]byte, []byte, []byte) {
	if len(l) < 2 {
		return nil, nil, nil
	}
	if l[0] != '_' {
		return nil, nil, nil
	}
	class, l := extractInside(l[1:], '(', ')')
	idx := bytes.IndexByte(l, '_')
	if idx == -1 {
		return nil, nil, nil
	}
	inside := l[:idx]
	rest := l[idx+1:]
	return inside, class, rest
}

// @$inside@$rest
func isCode(l []byte) ([]byte, []byte) {
	return extractInside(l, '@', '@')
}

// -$inside-$rest
func isDel(l []byte) ([]byte, []byte) {
	return extractInside(l, '-', '-')
}

// +$inside+$rest
func isIns(l []byte) ([]byte, []byte) {
	return extractInside(l, '+', '+')
}

// ^$inside^$rest
func isSup(l []byte) ([]byte, []byte) {
	return extractInside(l, '^', '^')
}

// ~$inside~$rest
func isSub(l []byte) ([]byte, []byte) {
	return extractInside(l, '~', '~')
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

// __$italic__$rest
func isItalic(l []byte) ([]byte, []byte) {
	return is2Byte(l, '_')
}

// **$bold**$rest
func isBold(l []byte) ([]byte, []byte) {
	return is2Byte(l, '*')
}

func isCite(l []byte) ([]byte, []byte) {
	return is2Byte(l, '?')
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
	title, l := extractInside(l, '"', '"')
	if title == nil || len(l) < 1 || l[0] != ':' {
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

func startsWith(l, prefix []byte) []byte {
	if bytes.HasPrefix(l, prefix) {
		return l[len(prefix):]
	}
	return nil
}

// notextile. $rest
func isNoTextile(l []byte) []byte {
	return startsWith(l, []byte("notextile. "))
}

// bq. $rest
func isBlockQuote(l []byte) []byte {
	return startsWith(l, []byte("bq. "))
}

// p. $rest
func isP(l []byte) []byte {
	return startsWith(l, []byte("p. "))
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

func (p *TextileParser) serEscaped(l []byte) {
	if isHtmlLine(l) {
		p.out.Write(l)
		return
	}

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
	p.serEscaped(before)
	p.serTagStartWithOptClass(tag, nil)
	p.serLine(inside)
	p.serTagEnd(tag)
	p.serLine(rest)
}

func (p *TextileParser) serTagWithOptClass(before, inside, class, rest []byte, tag string) {
	p.out.Write(before) // TODO: escaped?
	p.serTagStartWithOptClass(tag, class)
	p.serLine(inside)
	p.serTagEnd(tag)
	p.serLine(rest)
}

func (p *TextileParser) serTagWithOptStyle(before, inside, style, rest []byte, tag string) {
	p.serEscaped(before) // TODO: escaped?
	p.serTagStartWithOptStyle(tag, style)
	p.serLine(inside)
	p.serTagEnd(tag)
	p.serLine(rest)
}

func (p *TextileParser) serSpanWithLang(before, lang, inside, rest []byte) {
	p.serEscaped(before)
	p.out.WriteString(fmt.Sprintf(`<span lang="%s">`, string(lang)))
	p.serLine(inside)
	p.out.WriteString("</span>")
	p.serLine(rest)
}

func (p *TextileParser) serUrl(before, title, url, rest []byte) {
	p.serEscaped(before)
	p.out.WriteString(fmt.Sprintf(`<a href="%s">`, string(url)))
	p.serEscaped(title)
	p.out.WriteString("</a>")
	p.serLine(rest)
}

func (p *TextileParser) serCode(before, inside, rest []byte) {
	p.serEscaped(before)
	p.out.WriteString(fmt.Sprintf(`<code>%s</code>`, string(inside)))
	p.serLine(rest)
}

func (p *TextileParser) serImg(before []byte, imgSrc []byte, alt []byte, style int, url []byte, rest []byte) {
	p.serEscaped(before)
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

func (p *TextileParser) serNoTextile(s []byte) {
	p.out.Write(s)
}

func (p *TextileParser) serP(s []byte) {
	p.out.WriteString(fmt.Sprintf("\t<p>%s</p>", string(s)))
}

func (p *TextileParser) serBlockQuote(s []byte) {
	p.out.WriteString("\t<blockquote>\n\t")
	p.serP(s)
	p.out.WriteString("\n\t</blockquote>")
}

func (p *TextileParser) serHLine(n int, inside []byte) {
	p.out.WriteString(fmt.Sprintf("\t<h%d>", n))
	p.out.Write(inside) // TODO: escape?
	p.out.WriteString(fmt.Sprintf("</h%d>", n))
}

func (p *TextileParser) serLine(l []byte) {
	for i := 0; i < len(l); i++ {
		b := l[i]
		if b == '_' {
			if inside, rest := isItalic(l[i:]); inside != nil {
				p.serTag(l[:i], inside, rest, "i")
				return
			}
			if inside, class, rest := isEmWithOptClass(l[i:]); inside != nil {
				p.serTagWithOptClass(l[:i], inside, class, rest, "em")
				return
			}
		} else if b == '*' {
			if inside, rest := isBold(l[i:]); inside != nil {
				p.serTag(l[:i], inside, rest, "b")
				return
			}
			if inside, style, rest := isStrongWithOptStyle(l[i:]); inside != nil {
				p.serTagWithOptStyle(l[:i], inside, style, rest, "strong")
				return
			}
		} else if b == '%' {
			if inside, lang, rest := isSpanWithLang(l[i:]); inside != nil {
				p.serSpanWithLang(l[:i], lang, inside, rest)
				return
			}
			if inside, style, rest := isSpanWithOptStyle(l[i:]); inside != nil {
				p.serTagWithOptStyle(l[:i], inside, style, rest, "span")
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
		} else if b == '@' {
			if inside, rest := isCode(l[i:]); inside != nil {
				p.serCode(l[:i], inside, rest)
				return
			}
		} else if b == '?' {
			if inside, rest := isCite(l[i:]); inside != nil {
				p.serTag(l[:i], inside, rest, "cite")
				return
			}
		} else if b == '-' {
			if inside, rest := isDel(l[i:]); inside != nil {
				p.serTag(l[:i], inside, rest, "del")
				return
			}
		} else if b == '+' {
			if inside, rest := isIns(l[i:]); inside != nil {
				p.serTag(l[:i], inside, rest, "ins")
				return
			}
		} else if b == '^' {
			if inside, rest := isSup(l[i:]); inside != nil {
				p.serTag(l[:i], inside, rest, "sup")
				return
			}
		} else if b == '~' {
			if inside, rest := isSub(l[i:]); inside != nil {
				p.serTag(l[:i], inside, rest, "sub")
				return
			}
		}
	}
	p.serEscaped(l)
}

func (p *TextileParser) serLines(lines [][]byte) {
	for i, l := range lines {
		p.serLine(l)
		if i != len(lines)-1 {
			if p.r.isXhtml() {
				p.out.WriteString("<br />")
			} else {
				p.out.WriteString("<br>")
			}
			p.out.Write(newline)
		}
	}
}

func (p *TextileParser) serParagraph(lines [][]byte) {
	if len(lines) > 0 {
		l := lines[0]
		//fmt.Printf("serParagraph(): %s\n", string(l))
		if n, inside := isHLine(l); n != -1 {
			//fmt.Printf("serParagraph(): h%d '%s'\n", n, string(inside))
			p.serHLine(n, inside)
			if len(lines) > 1 {
				p.serParagraph(lines[1:])
			}
			return
		}
	}
	if len(lines) == 1 {
		l := lines[0]
		if rest := isNoTextile(l); rest != nil {
			p.serNoTextile(rest)
			return
		}
		if rest := isP(l); rest != nil {
			p.serP(rest)
			return
		}
		if rest := isBlockQuote(l); rest != nil {
			p.serBlockQuote(rest)
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
