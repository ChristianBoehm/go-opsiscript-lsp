package lsp

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ChristianBoehm/go-opsiscript-lsp/internal/ast"
	"github.com/ChristianBoehm/go-opsiscript-lsp/internal/files"
	"github.com/ChristianBoehm/go-opsiscript-lsp/internal/parser"
)

func loadWorkspaceDocuments(uri, text string) []*ast.Document {
	root := parser.Parse(uri, text)
	documents := []*ast.Document{root}
	queue := []*ast.Document{root}
	seen := map[string]struct{}{uri: {}}

	for len(queue) > 0 {
		document := queue[0]
		queue = queue[1:]

		path, err := files.URIToPath(document.URI)
		if err != nil || path == "" {
			continue
		}
		baseDir := filepath.Dir(path)

		for _, include := range document.Includes {
			includePath := files.ResolveIncludePath(baseDir, include.Target)
			if includePath == "" {
				continue
			}

			includeURI := files.PathToURI(includePath)
			if _, ok := seen[includeURI]; ok {
				continue
			}

			content, err := os.ReadFile(includePath)
			if err != nil {
				document.Diagnostics = append(document.Diagnostics, ast.Diagnostic{
					URI:      document.URI,
					Range:    include.Range,
					Severity: ast.SeverityWarning,
					Source:   "opsiscript-lsp",
					Message:  fmt.Sprintf("unable to load %s %q", include.Command, include.Target),
				})
				continue
			}

			child := parser.Parse(includeURI, string(content))
			documents = append(documents, child)
			queue = append(queue, child)
			seen[includeURI] = struct{}{}
		}
	}

	return documents
}
