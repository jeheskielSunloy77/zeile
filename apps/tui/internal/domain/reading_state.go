package domain

import "time"

type ReadingMode string

const (
	ReadingModeEPUB      ReadingMode = "epub"
	ReadingModePDFText   ReadingMode = "pdf_text"
	ReadingModePDFLayout ReadingMode = "pdf_layout"
)

type Locator struct {
	Offset       int `json:"offset,omitempty"`
	PageIndex    int `json:"page_index,omitempty"`
	SectionIndex int `json:"section_index,omitempty"`
}

type ReadingState struct {
	BookID          string
	Mode            ReadingMode
	Locator         Locator
	ProgressPercent float64
	UpdatedAt       time.Time
	IsFinished      bool
}
