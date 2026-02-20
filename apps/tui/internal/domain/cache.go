package domain

type EPUBCache struct {
	Title    string   `json:"title"`
	Author   string   `json:"author"`
	Sections []string `json:"sections"`
}

type PDFCache struct {
	Title       string   `json:"title"`
	Author      string   `json:"author"`
	Pages       []string `json:"pages"`
	LayoutPages []string `json:"layout_pages,omitempty"`
}
