package epub

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	stdhtml "html"
	"io"
	"path"
	"strings"

	xhtml "golang.org/x/net/html"

	"github.com/zeile/tui/internal/domain"
)

type containerDoc struct {
	Rootfiles struct {
		Items []struct {
			FullPath string `xml:"full-path,attr"`
		} `xml:"rootfile"`
	} `xml:"rootfiles"`
}

type packageDoc struct {
	Metadata struct {
		Titles   []string `xml:"title"`
		Creators []string `xml:"creator"`
	} `xml:"metadata"`
	Manifest struct {
		Items []struct {
			ID   string `xml:"id,attr"`
			Href string `xml:"href,attr"`
		} `xml:"item"`
	} `xml:"manifest"`
	Spine struct {
		Itemrefs []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"itemref"`
	} `xml:"spine"`
}

func Extract(ctx context.Context, pathToEPUB string) (domain.EPUBCache, error) {
	if err := ctx.Err(); err != nil {
		return domain.EPUBCache{}, err
	}

	archive, err := zip.OpenReader(pathToEPUB)
	if err != nil {
		return domain.EPUBCache{}, fmt.Errorf("open epub: %w", err)
	}
	defer archive.Close()

	files := make(map[string]*zip.File, len(archive.File))
	for _, file := range archive.File {
		files[normalize(file.Name)] = file
	}

	containerBytes, err := readZipFile(files, "META-INF/container.xml")
	if err != nil {
		return domain.EPUBCache{}, fmt.Errorf("read container.xml: %w", err)
	}

	var container containerDoc
	if err := xml.Unmarshal(containerBytes, &container); err != nil {
		return domain.EPUBCache{}, fmt.Errorf("decode container.xml: %w", err)
	}

	if len(container.Rootfiles.Items) == 0 {
		return domain.EPUBCache{}, fmt.Errorf("epub has no rootfile entries")
	}

	opfPath := normalize(container.Rootfiles.Items[0].FullPath)
	opfBytes, err := readZipFile(files, opfPath)
	if err != nil {
		return domain.EPUBCache{}, fmt.Errorf("read package document: %w", err)
	}

	var pkg packageDoc
	if err := xml.Unmarshal(opfBytes, &pkg); err != nil {
		return domain.EPUBCache{}, fmt.Errorf("decode package document: %w", err)
	}

	manifest := map[string]string{}
	for _, item := range pkg.Manifest.Items {
		manifest[item.ID] = item.Href
	}

	opfDir := path.Dir(opfPath)
	sections := make([]string, 0, len(pkg.Spine.Itemrefs))
	for _, itemRef := range pkg.Spine.Itemrefs {
		if err := ctx.Err(); err != nil {
			return domain.EPUBCache{}, err
		}

		href, ok := manifest[itemRef.IDRef]
		if !ok {
			continue
		}

		chapterPath := normalize(path.Join(opfDir, href))
		chapterBytes, err := readZipFile(files, chapterPath)
		if err != nil {
			continue
		}

		chapterText := extractText(chapterBytes)
		if chapterText != "" {
			sections = append(sections, chapterText)
		}
	}

	if len(sections) == 0 {
		return domain.EPUBCache{}, fmt.Errorf("no readable text extracted from epub")
	}

	cache := domain.EPUBCache{
		Title:    firstNonEmpty(pkg.Metadata.Titles...),
		Author:   firstNonEmpty(pkg.Metadata.Creators...),
		Sections: sections,
	}

	if cache.Title == "" {
		cache.Title = "Untitled EPUB"
	}
	if cache.Author == "" {
		cache.Author = "Unknown"
	}

	return cache, nil
}

func readZipFile(files map[string]*zip.File, name string) ([]byte, error) {
	file, ok := files[normalize(name)]
	if !ok {
		return nil, fmt.Errorf("%s not found", name)
	}

	reader, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", name, err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	return content, nil
}

func extractText(content []byte) string {
	tokenizer := xhtml.NewTokenizer(bytes.NewReader(content))
	parts := make([]string, 0, 64)

	for {
		typeToken := tokenizer.Next()
		switch typeToken {
		case xhtml.ErrorToken:
			if tokenizer.Err() == io.EOF {
				joined := strings.Join(parts, " ")
				joined = strings.ReplaceAll(joined, "\u00a0", " ")
				joined = normalizeWhitespace(joined)
				return joined
			}
			return ""
		case xhtml.StartTagToken, xhtml.EndTagToken:
			token := tokenizer.Token()
			if isBlockTag(token.Data) {
				parts = append(parts, "\n")
			}
		case xhtml.TextToken:
			text := strings.TrimSpace(stdhtml.UnescapeString(string(tokenizer.Text())))
			if text != "" {
				parts = append(parts, text)
			}
		}
	}
}

func isBlockTag(tag string) bool {
	switch strings.ToLower(tag) {
	case "p", "div", "section", "article", "header", "footer", "h1", "h2", "h3", "h4", "h5", "h6", "li", "br", "tr":
		return true
	default:
		return false
	}
}

func normalizeWhitespace(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	lines := strings.Split(value, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if len(cleaned) > 0 && cleaned[len(cleaned)-1] != "" {
				cleaned = append(cleaned, "")
			}
			continue
		}
		cleaned = append(cleaned, strings.Join(strings.Fields(line), " "))
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalize(value string) string {
	cleaned := path.Clean(strings.ReplaceAll(value, "\\", "/"))
	cleaned = strings.TrimPrefix(cleaned, "./")
	return cleaned
}
