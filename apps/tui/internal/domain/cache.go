package domain

type EPUBCache struct {
	Title                     string              `json:"title"`
	Author                    string              `json:"author"`
	Sections                  []string            `json:"sections"`
	SectionChapterLineIndexes [][]int             `json:"section_chapter_line_indexes,omitempty"`
	SectionInlineStyles       [][]InlineStyleSpan `json:"section_inline_styles,omitempty"`
}

type PDFCache struct {
	Title       string   `json:"title"`
	Author      string   `json:"author"`
	Pages       []string `json:"pages"`
	LayoutPages []string `json:"layout_pages,omitempty"`
}

type InlineStyle uint16

const (
	InlineStyleBold InlineStyle = 1 << iota
	InlineStyleItalic
	InlineStyleUnderline
	InlineStyleMark
	InlineStyleSmall
	InlineStyleSub
	InlineStyleSup
	InlineStyleCode
)

type InlineStyleSpan struct {
	LineIndex int         `json:"line_index"`
	StartWord int         `json:"start_word"`
	EndWord   int         `json:"end_word"`
	Style     InlineStyle `json:"style"`
}
