package textiler

import (
	"bytes"
	"fmt"
)

var newline = []byte{'\n'}

func lastByte(d []byte) byte {
	return d[len(d)-1]
}

func slice(d []byte, start, end int) []byte {
	if end > 0 {
		return d[start:end]
	}
	end = len(d) -1 + end
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
	case '"':
		return []byte("&quot;")
	case '&':
		return []byte("&amp;")
	case '<':
		return []byte("&lt;")
	case '>':
		return []byte("&gt;")
	}
	return nil
}

func serHtmlEscaped(d []byte, out *bytes.Buffer) {
	for _, b := range d {
		if esc := needsHtmlEscaping(b); esc != nil {
			out.Write(esc)
		} else {
			out.WriteByte(b)
		}
	}	
}

func serHtmlEscapedLines(lines [][]byte, out *bytes.Buffer) {
	for i, l := range lines {
		serHtmlEscaped(l, out)
		if i != len(lines) - 1 {
			out.Write(newline)
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

func serLines(lines [][]byte, out *bytes.Buffer) {
	for i, l := range lines {
		out.Write(l)
		if i != len(lines) - 1 {
			// TODO: in xhtml mode, output "<br />"
			out.WriteString("<br>")
			out.Write(newline)
		}
	}
}

func serParagraph(lines [][]byte, out *bytes.Buffer) {
	out.WriteString("\t<p>")
	serLines(lines, out)
	out.WriteString("</p>")
}

func serHtmlParagraph(lines [][]byte, out *bytes.Buffer) {
	out.Write(lines[0])
	out.Write(newline)
	middleLines := lines[1:len(lines)-1]
	serHtmlEscapedLines(middleLines, out)
	out.Write(newline)
	out.Write(lines[len(lines)-1])
}

func serParagraphs(paragraphs [][][]byte, out *bytes.Buffer) {
	for i, para := range paragraphs {
		if i != 0 {
			out.Write(newline)
		}
		if isHtmlParagraph(para) {
			serHtmlParagraph(para, out)
		} else {
			serParagraph(para, out)
		}
		if i != len(paragraphs) - 1 {
			out.Write(newline)
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
			currPara = make([][]byte,0)
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

func ToHtml(d []byte, flagDumpLines, flagDumpParagraphs bool) []byte {
	var out bytes.Buffer
	lines := splitIntoLines(d)

	if flagDumpLines {
		var buf bytes.Buffer
		dumpLines(lines, &buf)
		fmt.Printf("%s", string(buf.Bytes()))
		return nil
	}

	paragraphs := groupIntoParagraphs(lines)
	if flagDumpParagraphs {
		var buf bytes.Buffer
		dumpParagraphs(paragraphs, &buf)
		fmt.Printf("%s", string(buf.Bytes()))
		return nil
	}

	serParagraphs(paragraphs, &out)
	return out.Bytes()
}

func ToXhtml(d []byte, flagDumpLines, flagDumpParagraphs bool) []byte {
	return ToHtml(d, flagDumpLines, flagDumpParagraphs)
}
