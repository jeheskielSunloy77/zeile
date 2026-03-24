package preprocess

import "path/filepath"

const (
	epubCacheFile = "epub_cache.json"
)

func epubCachePath(cacheDir string) string {
	return filepath.Join(cacheDir, epubCacheFile)
}
