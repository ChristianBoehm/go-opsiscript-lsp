package files

import (
	"path/filepath"
	"regexp"
	"strings"
)

var scriptPathPattern = regexp.MustCompile(`(?i)%scriptpath%`)

func ResolveIncludePath(baseDir, target string) string {
	resolved := strings.TrimSpace(target)
	if resolved == "" {
		return ""
	}

	resolved = scriptPathPattern.ReplaceAllString(resolved, filepath.ToSlash(baseDir))
	resolved = strings.ReplaceAll(resolved, "\\", "/")
	resolved = filepath.FromSlash(resolved)

	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(baseDir, resolved)
	}

	return filepath.Clean(resolved)
}
