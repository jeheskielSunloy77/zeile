package reader

import (
	"sort"
	"strings"
)

type TokenType int

const (
	TokenWord TokenType = iota
	TokenNewline
)

type TextStyle uint16

const (
	TextStyleBold TextStyle = 1 << iota
	TextStyleItalic
	TextStyleUnderline
	TextStyleMark
	TextStyleSmall
	TextStyleSub
	TextStyleSup
	TextStyleCode
)

type Token struct {
	Type  TokenType
	Value string
	Style TextStyle
}

type TextDocument struct {
	Tokens           []Token
	plain            string
	tokenStartOffset []int
}

type TextPagination struct {
	Pages          []string
	PageStarts     []int
	PageLineStarts [][]int
	PageLineRanges [][]TokenRange
}

func NewTextDocument(text string) TextDocument {
	return NewTextDocumentWithStyles(text, nil)
}

func NewTextDocumentWithStyles(text string, tokenStyles map[int]TextStyle) TextDocument {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	tokens := make([]Token, 0, len(text)/4)
	tokenIndex := 0
	lines := strings.Split(text, "\n")
	for index, line := range lines {
		words := strings.Fields(line)
		for _, word := range words {
			tokens = append(tokens, Token{Type: TokenWord, Value: word, Style: tokenStyles[tokenIndex]})
			tokenIndex++
		}
		if index < len(lines)-1 {
			tokens = append(tokens, Token{Type: TokenNewline})
			tokenIndex++
		}
	}

	if len(tokens) == 0 {
		tokens = append(tokens, Token{Type: TokenWord, Value: "", Style: tokenStyles[tokenIndex]})
	}

	plainBuilder := strings.Builder{}
	starts := make([]int, 0, len(tokens))
	atLineStart := true
	for _, token := range tokens {
		starts = append(starts, plainBuilder.Len())
		switch token.Type {
		case TokenNewline:
			plainBuilder.WriteByte('\n')
			atLineStart = true
		case TokenWord:
			if !atLineStart {
				plainBuilder.WriteByte(' ')
			}
			plainBuilder.WriteString(token.Value)
			atLineStart = false
		}
	}

	return TextDocument{
		Tokens:           tokens,
		plain:            plainBuilder.String(),
		tokenStartOffset: starts,
	}
}

type TokenRange struct {
	Start int
	End   int
}

func (d TextDocument) TokenCount() int {
	return len(d.Tokens)
}

func (d TextDocument) SearchTokenOffsets(query string) []int {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}

	lowerText := strings.ToLower(d.plain)
	lowerQuery := strings.ToLower(query)
	results := make([]int, 0)
	seen := map[int]struct{}{}

	from := 0
	for {
		idx := strings.Index(lowerText[from:], lowerQuery)
		if idx < 0 {
			break
		}
		matchStart := from + idx
		tokenIndex := d.TokenIndexAtCharOffset(matchStart)
		if _, exists := seen[tokenIndex]; !exists {
			seen[tokenIndex] = struct{}{}
			results = append(results, tokenIndex)
		}
		from = matchStart + len(lowerQuery)
		if from >= len(lowerText) {
			break
		}
	}

	sort.Ints(results)
	return results
}

func (d TextDocument) TokenIndexAtCharOffset(offset int) int {
	if offset <= 0 {
		return 0
	}
	if len(d.tokenStartOffset) == 0 {
		return 0
	}

	index := sort.Search(len(d.tokenStartOffset), func(i int) bool {
		return d.tokenStartOffset[i] > offset
	}) - 1
	if index < 0 {
		return 0
	}
	if index >= len(d.tokenStartOffset) {
		return len(d.tokenStartOffset) - 1
	}
	return index
}

func (d TextDocument) Paginate(width, height int) TextPagination {
	return d.PaginateWithForcedPageStarts(width, height, nil)
}

func (d TextDocument) PaginateWithForcedPageStarts(width, height int, forcedStarts map[int]struct{}) TextPagination {
	if width < 20 {
		width = 20
	}
	if height < 5 {
		height = 5
	}

	type line struct {
		Text       string
		StartToken int
		EndToken   int
	}

	lines := make([]line, 0, len(d.Tokens)/3)
	currentLine := strings.Builder{}
	lineStartToken := -1
	lineEndToken := -1
	lineHasWords := false

	flushLine := func(forceEmpty bool, startToken int) {
		if lineHasWords {
			lines = append(lines, line{
				Text:       currentLine.String(),
				StartToken: lineStartToken,
				EndToken:   lineEndToken + 1,
			})
			currentLine.Reset()
			lineStartToken = -1
			lineEndToken = -1
			lineHasWords = false
			return
		}
		if forceEmpty {
			lines = append(lines, line{Text: "", StartToken: startToken, EndToken: startToken})
		}
	}

	for idx, token := range d.Tokens {
		switch token.Type {
		case TokenNewline:
			flushLine(true, idx)
			lineStartToken = idx + 1
		case TokenWord:
			if !lineHasWords {
				currentLine.WriteString(token.Value)
				lineStartToken = idx
				lineEndToken = idx
				lineHasWords = true
				continue
			}

			candidate := currentLine.String() + " " + token.Value
			if len([]rune(candidate)) <= width {
				currentLine.Reset()
				currentLine.WriteString(candidate)
				lineEndToken = idx
				continue
			}

			flushLine(false, idx)
			currentLine.WriteString(token.Value)
			lineStartToken = idx
			lineEndToken = idx
			lineHasWords = true
		}
	}
	flushLine(false, len(d.Tokens)-1)

	if len(lines) == 0 {
		lines = append(lines, line{Text: "", StartToken: 0, EndToken: 0})
	}

	pages := make([]string, 0, (len(lines)/height)+1)
	pageStarts := make([]int, 0, (len(lines)/height)+1)
	pageLineStarts := make([][]int, 0, (len(lines)/height)+1)
	pageLineRanges := make([][]TokenRange, 0, (len(lines)/height)+1)

	pageTexts := make([]string, 0, height)
	pageStartsForLines := make([]int, 0, height)
	pageRangesForLines := make([]TokenRange, 0, height)

	flushPage := func() {
		if len(pageTexts) == 0 {
			return
		}
		pages = append(pages, strings.Join(pageTexts, "\n"))
		pageLineStarts = append(pageLineStarts, append([]int(nil), pageStartsForLines...))
		pageLineRanges = append(pageLineRanges, append([]TokenRange(nil), pageRangesForLines...))

		start := pageStartsForLines[0]
		if start < 0 {
			start = 0
		}
		pageStarts = append(pageStarts, start)

		pageTexts = pageTexts[:0]
		pageStartsForLines = pageStartsForLines[:0]
		pageRangesForLines = pageRangesForLines[:0]
	}

	for _, item := range lines {
		if len(pageTexts) > 0 {
			if _, ok := forcedStarts[item.StartToken]; ok {
				flushPage()
			}
		}
		if len(pageTexts) >= height {
			flushPage()
		}
		pageTexts = append(pageTexts, item.Text)
		pageStartsForLines = append(pageStartsForLines, item.StartToken)
		pageRangesForLines = append(pageRangesForLines, TokenRange{Start: item.StartToken, End: item.EndToken})
	}
	flushPage()

	return TextPagination{
		Pages:          pages,
		PageStarts:     pageStarts,
		PageLineStarts: pageLineStarts,
		PageLineRanges: pageLineRanges,
	}
}

func (p TextPagination) PageForOffset(offset int) int {
	if len(p.PageStarts) == 0 || offset <= 0 {
		return 0
	}

	index := sort.Search(len(p.PageStarts), func(i int) bool {
		return p.PageStarts[i] > offset
	}) - 1
	if index < 0 {
		return 0
	}
	if index >= len(p.PageStarts) {
		return len(p.PageStarts) - 1
	}
	return index
}

func (p TextPagination) OffsetForPage(page int) int {
	if len(p.PageStarts) == 0 {
		return 0
	}
	if page < 0 {
		return p.PageStarts[0]
	}
	if page >= len(p.PageStarts) {
		return p.PageStarts[len(p.PageStarts)-1]
	}
	return p.PageStarts[page]
}

func ProgressPercent(offset, totalTokens int) float64 {
	if totalTokens <= 0 {
		return 0
	}
	if offset < 0 {
		offset = 0
	}
	if offset > totalTokens {
		offset = totalTokens
	}
	return float64(offset) / float64(totalTokens) * 100
}
