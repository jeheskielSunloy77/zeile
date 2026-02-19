package domain

func DefaultModeForFormat(format BookFormat) ReadingMode {
	switch format {
	case BookFormatPDF:
		return ReadingModePDFText
	default:
		return ReadingModeEPUB
	}
}
