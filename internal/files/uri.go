package files

import (
	"net/url"
	"path/filepath"
	"strings"
)

func URIToPath(uri string) (string, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return "", err
	}

	if parsed.Scheme != "file" {
		return "", nil
	}

	return filepath.FromSlash(parsed.Path), nil
}

func PathToURI(path string) string {
	value := filepath.ToSlash(path)
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return "file://" + value
}
