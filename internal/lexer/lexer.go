package lexer

import (
	"strings"

	"github.com/ChristianBoehm/go-opsiscript-lsp/internal/ast"
)

type Line struct {
	Number  int
	Text    string
	Trimmed string
}

type Result struct {
	Lines       []Line
	Diagnostics []ast.Diagnostic
}

func Lex(text string) Result {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	rawLines := strings.Split(normalized, "\n")

	result := Result{
		Lines: make([]Line, 0, len(rawLines)),
	}

	for lineNumber, textLine := range rawLines {
		trimmed := strings.TrimSpace(textLine)
		result.Lines = append(result.Lines, Line{
			Number:  lineNumber,
			Text:    textLine,
			Trimmed: trimmed,
		})

		if isComment(trimmed) {
			continue
		}

		if inSingle, inDouble := unmatchedQuotes(textLine); inDouble {
			result.Diagnostics = append(result.Diagnostics, ast.Diagnostic{
				URI:      "",
				Range:    ast.NewRange(lineNumber, 0, len(textLine)),
				Severity: ast.SeverityError,
				Source:   "opsiscript-lsp",
				Message:  "unclosed double-quoted string",
			})
		} else if inSingle {
			result.Diagnostics = append(result.Diagnostics, ast.Diagnostic{
				URI:      "",
				Range:    ast.NewRange(lineNumber, 0, len(textLine)),
				Severity: ast.SeverityError,
				Source:   "opsiscript-lsp",
				Message:  "unclosed single-quoted string",
			})
		}
	}

	return result
}

func isComment(trimmed string) bool {
	return strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "//")
}

func unmatchedQuotes(line string) (inSingle bool, inDouble bool) {
	for _, r := range line {
		switch {
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		}
	}

	return inSingle, inDouble
}
