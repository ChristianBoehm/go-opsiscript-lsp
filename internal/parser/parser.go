package parser

import (
	"regexp"
	"strings"

	"github.com/ChristianBoehm/go-opsiscript-lsp/internal/ast"
	"github.com/ChristianBoehm/go-opsiscript-lsp/internal/lexer"
	"github.com/ChristianBoehm/go-opsiscript-lsp/internal/symbols"
)

var (
	sectionCallPrefixesExpr  = symbols.SectionCallPrefixesPattern()
	sectionHeaderPattern     = regexp.MustCompile(`^\s*\[([A-Za-z][A-Za-z0-9_]*)\]\s*$`)
	variablePattern          = regexp.MustCompile(`\$[A-Za-z_][A-Za-z0-9_]*\$`)
	variableDeclPattern      = regexp.MustCompile(`(?i)^\s*(DefVar|DefStringList|DefStringlist)\s+(\$[A-Za-z_][A-Za-z0-9_]*\$)`)
	defFuncPattern           = regexp.MustCompile(`(?i)^\s*DefFunc\s+([A-Za-z_][A-Za-z0-9_]*)\s*\((.*)\)\s*(?::\s*([A-Za-z_][A-Za-z0-9_]*))?`)
	endFuncPattern           = regexp.MustCompile(`(?i)^\s*EndFunc\b`)
	includePattern           = regexp.MustCompile(`(?i)^\s*(include_insert|include_append|importlib)\s+("([^"]*)"|'([^']*)'|([^\s;]+))`)
	sectionCallLinePattern   = regexp.MustCompile(`(?i)^\s*((` + sectionCallPrefixesExpr + `)_[A-Za-z0-9_]+)\b(.*)$`)
	sectionCallStringPattern = regexp.MustCompile(`(?i)(executeSection|getOutStreamFromSection|getReturnListFromSection)\s*\(\s*["']([^"']+)["']\s*\)`)
	modifierTokenPattern     = regexp.MustCompile(`(?i)\bwinst\b|/[A-Za-z0-9]+(?:\s+[^/\s][^\s]*)?`)
	functionCallPattern      = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	wordPattern              = regexp.MustCompile(`\b[A-Za-z_][A-Za-z0-9_]*\b`)
	commandHeadPattern       = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)`)
	sectionCallPattern       = regexp.MustCompile(`(?i)\b((` + sectionCallPrefixesExpr + `)_[A-Za-z0-9_]+)\b`)
	constantPattern          = regexp.MustCompile(`%[A-Za-z_][A-Za-z0-9_]*%`)
	primarySectionPattern    = regexp.MustCompile(`(?i)^(Initial|Actions|ProfileActions)$`)
)

func Parse(uri, text string) *ast.Document {
	lexed := lexer.Lex(text)
	document := &ast.Document{
		URI:         uri,
		Text:        text,
		Lines:       make([]string, 0, len(lexed.Lines)),
		Diagnostics: append([]ast.Diagnostic{}, lexed.Diagnostics...),
	}
	for index := range document.Diagnostics {
		document.Diagnostics[index].URI = uri
	}

	currentFunction := ""

	for _, line := range lexed.Lines {
		document.Lines = append(document.Lines, line.Text)
		if line.Trimmed == "" || strings.HasPrefix(line.Trimmed, ";") || strings.HasPrefix(line.Trimmed, "//") {
			continue
		}

		if matches := defFuncPattern.FindStringSubmatchIndex(line.Text); matches != nil {
			functionName := line.Text[matches[2]:matches[3]]
			returnType := ""
			if matches[6] >= 0 {
				returnType = line.Text[matches[6]:matches[7]]
			}

			document.Functions = append(document.Functions, ast.FunctionDecl{
				URI:            uri,
				Name:           functionName,
				NormalizedName: symbols.NormalizeName(functionName),
				ReturnType:     returnType,
				Range:          ast.NewRange(line.Number, matches[0], matches[1]),
				NameRange:      ast.NewRange(line.Number, matches[2], matches[3]),
			})

			for _, capture := range variablePattern.FindAllStringSubmatchIndex(line.Text, -1) {
				name := line.Text[capture[0]:capture[1]]
				document.Variables = append(document.Variables, ast.VariableDecl{
					URI:            uri,
					Name:           name,
					NormalizedName: symbols.NormalizeName(name),
					Kind:           "parameter",
					Scope:          symbols.NormalizeName(functionName),
					Range:          ast.NewRange(line.Number, capture[0], capture[1]),
					NameRange:      ast.NewRange(line.Number, capture[0], capture[1]),
				})
			}

			currentFunction = symbols.NormalizeName(functionName)
			continue
		}

		if endFuncPattern.MatchString(line.Text) {
			currentFunction = ""
			continue
		}

		if matches := sectionHeaderPattern.FindStringSubmatchIndex(line.Text); matches != nil {
			sectionName := line.Text[matches[2]:matches[3]]
			kind := "Generic section"
			prefix := ""
			switch {
			case primarySectionPattern.MatchString(sectionName):
				kind = "Primary section"
			default:
				if typedPrefix, detail, ok := symbols.TypedSectionKind(sectionName); ok {
					prefix = typedPrefix
					kind = detail
				}
			}

			document.Sections = append(document.Sections, ast.Section{
				URI:            uri,
				Name:           sectionName,
				NormalizedName: symbols.NormalizeName(sectionName),
				Prefix:         prefix,
				Kind:           kind,
				Range:          ast.NewRange(line.Number, matches[0], matches[1]),
				NameRange:      ast.NewRange(line.Number, matches[2], matches[3]),
			})
			continue
		}

		if strings.HasPrefix(line.Trimmed, "[") {
			document.Diagnostics = append(document.Diagnostics, ast.Diagnostic{
				URI:      uri,
				Range:    ast.NewRange(line.Number, 0, len(line.Text)),
				Severity: ast.SeverityError,
				Source:   "opsiscript-lsp",
				Message:  "malformed section header",
			})
			continue
		}

		if matches := includePattern.FindStringSubmatchIndex(line.Text); matches != nil {
			target := includeTarget(line.Text, matches)
			document.Includes = append(document.Includes, ast.Include{
				Command: strings.ToLower(line.Text[matches[2]:matches[3]]),
				Target:  target,
				Range:   ast.NewRange(line.Number, matches[0], matches[1]),
			})
		}

		if matches := variableDeclPattern.FindStringSubmatchIndex(line.Text); matches != nil {
			name := line.Text[matches[4]:matches[5]]
			kind := strings.ToLower(line.Text[matches[2]:matches[3]])
			document.Variables = append(document.Variables, ast.VariableDecl{
				URI:            uri,
				Name:           name,
				NormalizedName: symbols.NormalizeName(name),
				Kind:           kind,
				Scope:          currentFunction,
				Range:          ast.NewRange(line.Number, matches[0], matches[1]),
				NameRange:      ast.NewRange(line.Number, matches[4], matches[5]),
			})
		}

		if matches := sectionCallLinePattern.FindStringSubmatchIndex(line.Text); matches != nil {
			appendSectionReference(document, uri, currentFunction, line.Number, line.Text, matches[2], matches[3], line.Text[matches[6]:matches[7]], "statement")
		}

		for _, matches := range sectionCallStringPattern.FindAllStringSubmatchIndex(line.Text, -1) {
			callSource := line.Text[matches[2]:matches[3]]
			callText := line.Text[matches[4]:matches[5]]
			appendStringSectionReference(document, uri, currentFunction, line.Number, line.Text, callSource, callText)
		}

		if matches := commandHeadPattern.FindStringSubmatchIndex(line.Text); matches != nil {
			name := line.Text[matches[2]:matches[3]]
			if _, ok := symbols.LookupCommand(name); ok {
				document.References = append(document.References, ast.Reference{
					URI:            uri,
					Name:           name,
					NormalizedName: symbols.NormalizeName(name),
					Kind:           ast.ReferenceCommand,
					Scope:          currentFunction,
					Range:          ast.NewRange(line.Number, matches[2], matches[3]),
				})
			}
		}

		for _, matches := range sectionCallPattern.FindAllStringSubmatchIndex(line.Text, -1) {
			name := line.Text[matches[0]:matches[1]]
			if containsReference(document.References, line.Number, matches[0], matches[1], ast.ReferenceSection) {
				continue
			}
			document.References = append(document.References, ast.Reference{
				URI:            uri,
				Name:           name,
				NormalizedName: symbols.NormalizeName(name),
				Kind:           ast.ReferenceSection,
				Scope:          currentFunction,
				CallStyle:      "inline",
				Range:          ast.NewRange(line.Number, matches[0], matches[1]),
			})
		}

		for _, matches := range variablePattern.FindAllStringSubmatchIndex(line.Text, -1) {
			name := line.Text[matches[0]:matches[1]]
			if existingDeclaration(document.Variables, line.Number, matches[0], matches[1]) {
				continue
			}
			document.References = append(document.References, ast.Reference{
				URI:            uri,
				Name:           name,
				NormalizedName: symbols.NormalizeName(name),
				Kind:           ast.ReferenceVariable,
				Scope:          currentFunction,
				Range:          ast.NewRange(line.Number, matches[0], matches[1]),
			})
		}

		for _, matches := range constantPattern.FindAllStringSubmatchIndex(line.Text, -1) {
			name := line.Text[matches[0]:matches[1]]
			document.References = append(document.References, ast.Reference{
				URI:            uri,
				Name:           name,
				NormalizedName: symbols.NormalizeName(name),
				Kind:           ast.ReferenceConstant,
				Scope:          currentFunction,
				Range:          ast.NewRange(line.Number, matches[0], matches[1]),
			})
		}

		for _, matches := range functionCallPattern.FindAllStringSubmatchIndex(line.Text, -1) {
			name := line.Text[matches[2]:matches[3]]
			document.References = append(document.References, ast.Reference{
				URI:            uri,
				Name:           name,
				NormalizedName: symbols.NormalizeName(name),
				Kind:           ast.ReferenceFunction,
				Scope:          currentFunction,
				Range:          ast.NewRange(line.Number, matches[2], matches[3]),
			})
		}

		for _, matches := range wordPattern.FindAllStringSubmatchIndex(line.Text, -1) {
			name := line.Text[matches[0]:matches[1]]
			if _, ok := symbols.LookupFunction(name); !ok {
				continue
			}
			if containsReference(document.References, line.Number, matches[0], matches[1], ast.ReferenceFunction) {
				continue
			}
			document.References = append(document.References, ast.Reference{
				URI:            uri,
				Name:           name,
				NormalizedName: symbols.NormalizeName(name),
				Kind:           ast.ReferenceFunction,
				Scope:          currentFunction,
				Range:          ast.NewRange(line.Number, matches[0], matches[1]),
			})
		}
	}

	return document
}

func includeTarget(line string, matches []int) string {
	switch {
	case matches[6] >= 0:
		return line[matches[6]:matches[7]]
	case matches[8] >= 0:
		return line[matches[8]:matches[9]]
	default:
		return line[matches[10]:matches[11]]
	}
}

func appendStringSectionReference(document *ast.Document, uri, scope string, lineNumber int, lineText, callSource, callText string) {
	matches := sectionCallLinePattern.FindStringSubmatchIndex(callText)
	if matches == nil {
		return
	}

	callStart := strings.Index(lineText, callText)
	if callStart < 0 {
		callStart = 0
	}

	appendSectionReference(document, uri, scope, lineNumber, lineText, callStart+matches[2], callStart+matches[3], callText[matches[6]:matches[7]], strings.ToLower(callSource))
}

func appendSectionReference(document *ast.Document, uri, scope string, lineNumber int, lineText string, start, end int, suffix, callStyle string) {
	if containsReference(document.References, lineNumber, start, end, ast.ReferenceSection) {
		return
	}

	name := lineText[start:end]
	document.References = append(document.References, ast.Reference{
		URI:            uri,
		Name:           name,
		NormalizedName: symbols.NormalizeName(name),
		Kind:           ast.ReferenceSection,
		Scope:          scope,
		CallStyle:      callStyle,
		Modifiers:      parseSectionModifiers(suffix),
		Range:          ast.NewRange(lineNumber, start, end),
	})
}

func parseSectionModifiers(text string) []string {
	matches := modifierTokenPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}

	modifiers := make([]string, 0, len(matches))
	for _, match := range matches {
		trimmed := strings.TrimSpace(match)
		if trimmed == "" {
			continue
		}
		modifiers = append(modifiers, trimmed)
	}
	return modifiers
}

func existingDeclaration(declarations []ast.VariableDecl, line, start, end int) bool {
	for _, declaration := range declarations {
		if declaration.NameRange.Start.Line == line &&
			declaration.NameRange.Start.Character == start &&
			declaration.NameRange.End.Character == end {
			return true
		}
	}

	return false
}

func containsReference(references []ast.Reference, line, start, end int, kind ast.ReferenceKind) bool {
	for _, reference := range references {
		if reference.Kind != kind {
			continue
		}
		if reference.Range.Start.Line == line &&
			reference.Range.Start.Character == start &&
			reference.Range.End.Character == end {
			return true
		}
	}

	return false
}
