package pdf

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode/utf8"

	"rsc.io/pdf"

	"github.com/zeile/tui/internal/domain"
)

func Extract(ctx context.Context, pathToPDF string) (domain.PDFCache, error) {
	if err := ctx.Err(); err != nil {
		return domain.PDFCache{}, err
	}

	reader, err := pdf.Open(pathToPDF)
	if err != nil {
		return domain.PDFCache{}, fmt.Errorf("open pdf: %w", err)
	}

	pageCount := reader.NumPage()
	if pageCount == 0 {
		return domain.PDFCache{}, fmt.Errorf("pdf has no pages")
	}

	pages := make([]string, 0, pageCount)
	layoutPages := make([]string, 0, pageCount)
	for pageNumber := 1; pageNumber <= pageCount; pageNumber++ {
		if err := ctx.Err(); err != nil {
			return domain.PDFCache{}, err
		}

		page := reader.Page(pageNumber)
		if page.V.IsNull() {
			pages = append(pages, "")
			layoutPages = append(layoutPages, "")
			continue
		}

		content := page.Content()
		textItems := content.Text
		sort.SliceStable(textItems, func(i, j int) bool {
			yDiff := math.Abs(textItems[i].Y - textItems[j].Y)
			if yDiff < 1.5 {
				return textItems[i].X < textItems[j].X
			}
			return textItems[i].Y > textItems[j].Y
		})

		lines := make([]string, 0, 64)
		currentLine := strings.Builder{}
		layoutLines := make([]string, 0, 64)
		layoutLine := strings.Builder{}
		lastY := 0.0
		hasLastY := false
		minX := 0.0
		maxX := 0.0
		hasRange := false
		const layoutWidth = 96
		for _, item := range textItems {
			if err := ctx.Err(); err != nil {
				return domain.PDFCache{}, err
			}

			chunk := strings.TrimSpace(item.S)
			if chunk == "" {
				continue
			}
			if !hasRange {
				minX = item.X
				maxX = item.X
				hasRange = true
			} else {
				if item.X < minX {
					minX = item.X
				}
				if item.X > maxX {
					maxX = item.X
				}
			}

			if hasLastY && math.Abs(item.Y-lastY) > 1.5 {
				line := strings.TrimSpace(currentLine.String())
				if line != "" {
					lines = append(lines, line)
				}
				currentLine.Reset()

				layoutText := strings.TrimRight(layoutLine.String(), " ")
				if layoutText != "" {
					layoutLines = append(layoutLines, layoutText)
				}
				layoutLine.Reset()
			}

			if currentLine.Len() > 0 {
				currentLine.WriteByte(' ')
			}
			currentLine.WriteString(chunk)

			col := 0
			if hasRange && maxX-minX > 0.001 {
				col = int(((item.X - minX) / (maxX - minX)) * float64(layoutWidth-1))
			}
			currentCols := utf8.RuneCountInString(layoutLine.String())
			padding := col - currentCols
			if currentCols > 0 && padding < 1 {
				padding = 1
			}
			if padding > 0 {
				layoutLine.WriteString(strings.Repeat(" ", padding))
			}
			layoutLine.WriteString(chunk)

			lastY = item.Y
			hasLastY = true
		}

		if line := strings.TrimSpace(currentLine.String()); line != "" {
			lines = append(lines, line)
		}
		if line := strings.TrimRight(layoutLine.String(), " "); line != "" {
			layoutLines = append(layoutLines, line)
		}

		pages = append(pages, strings.Join(lines, "\n"))
		layoutPages = append(layoutPages, strings.Join(layoutLines, "\n"))
	}

	cache := domain.PDFCache{
		Title:       "Untitled PDF",
		Author:      "Unknown",
		Pages:       pages,
		LayoutPages: layoutPages,
	}
	return cache, nil
}
