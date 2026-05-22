package analysis

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ChristianBoehm/go-opsiscript-lsp/internal/ast"
	"github.com/ChristianBoehm/go-opsiscript-lsp/internal/symbols"
)

var conditionalLinePattern = regexp.MustCompile(`(?i)^\s*(if|elseif|else|endif)\b`)
var logicalOperatorSuffixPattern = regexp.MustCompile(`(?i)\b(and|or)\s*$`)

func Analyze(document *ast.Document, index *symbols.Index) {
	if document == nil {
		return
	}

	for name, declarations := range index.Sections {
		if len(declarations) < 2 {
			continue
		}
		for _, duplicate := range declarations[1:] {
			if duplicate.URI != document.URI {
				continue
			}
			document.Diagnostics = append(document.Diagnostics, ast.Diagnostic{
				URI:      document.URI,
				Range:    duplicate.NameRange,
				Severity: ast.SeverityWarning,
				Source:   "opsiscript-lsp",
				Message:  fmt.Sprintf("duplicate section name %q", declarations[0].Name),
			})
		}
		delete(index.Sections, name)
	}

	for _, scopedVariables := range index.Variables {
		for _, declarations := range scopedVariables {
			if len(declarations) < 2 {
				continue
			}
			for _, duplicate := range declarations[1:] {
				if duplicate.URI != document.URI {
					continue
				}
				document.Diagnostics = append(document.Diagnostics, ast.Diagnostic{
					URI:      document.URI,
					Range:    duplicate.NameRange,
					Severity: ast.SeverityWarning,
					Source:   "opsiscript-lsp",
					Message:  fmt.Sprintf("duplicate variable declaration %q", declarations[0].Name),
				})
			}
		}
	}

	for _, reference := range document.References {
		if reference.Kind != ast.ReferenceSection {
			continue
		}
		if symbols.ResolveSection(index, reference.Name) != nil {
			continue
		}
		document.Diagnostics = append(document.Diagnostics, ast.Diagnostic{
			URI:      document.URI,
			Range:    reference.Range,
			Severity: ast.SeverityWarning,
			Source:   "opsiscript-lsp",
			Message:  fmt.Sprintf("unknown section call %q", reference.Name),
		})
	}

	analyzeConditionals(document)
}

type ifFrame struct {
	keywordRange ast.Range
	sawElse      bool
}

type delimiterState struct {
	quote        rune
	parentheses  int
	firstBadChar int
}

func analyzeConditionals(document *ast.Document) {
	var stack []ifFrame

	flushOpenIfs := func(line int) {
		for len(stack) > 0 {
			frame := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			document.Diagnostics = append(document.Diagnostics, ast.Diagnostic{
				URI:      document.URI,
				Range:    frame.keywordRange,
				Severity: ast.SeverityError,
				Source:   "opsiscript-lsp",
				Message:  "missing endif for if block",
			})
		}
		_ = line
	}

	for lineNumber, lineText := range document.Lines {
		trimmed := strings.TrimSpace(lineText)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "//") {
			continue
		}

		if strings.HasPrefix(trimmed, "[") {
			flushOpenIfs(lineNumber)
			continue
		}

		matches := conditionalLinePattern.FindStringSubmatchIndex(lineText)
		if matches == nil {
			continue
		}

		keyword := strings.ToLower(lineText[matches[2]:matches[3]])
		keywordRange := ast.NewRange(lineNumber, matches[2], matches[3])
		conditionText := lineText[matches[3]:]

		switch keyword {
		case "if":
			validateConditionalExpression(document, lineNumber, keywordRange, keyword, conditionText)
			stack = append(stack, ifFrame{keywordRange: keywordRange})
		case "elseif":
			validateConditionalExpression(document, lineNumber, keywordRange, keyword, conditionText)
			if len(stack) == 0 {
				document.Diagnostics = append(document.Diagnostics, ast.Diagnostic{
					URI:      document.URI,
					Range:    keywordRange,
					Severity: ast.SeverityError,
					Source:   "opsiscript-lsp",
					Message:  "elseif without matching if",
				})
				continue
			}
			if stack[len(stack)-1].sawElse {
				document.Diagnostics = append(document.Diagnostics, ast.Diagnostic{
					URI:      document.URI,
					Range:    keywordRange,
					Severity: ast.SeverityError,
					Source:   "opsiscript-lsp",
					Message:  "elseif after else in the same if block",
				})
			}
		case "else":
			if len(stack) == 0 {
				document.Diagnostics = append(document.Diagnostics, ast.Diagnostic{
					URI:      document.URI,
					Range:    keywordRange,
					Severity: ast.SeverityError,
					Source:   "opsiscript-lsp",
					Message:  "else without matching if",
				})
				continue
			}
			if stack[len(stack)-1].sawElse {
				document.Diagnostics = append(document.Diagnostics, ast.Diagnostic{
					URI:      document.URI,
					Range:    keywordRange,
					Severity: ast.SeverityError,
					Source:   "opsiscript-lsp",
					Message:  "duplicate else in the same if block",
				})
				continue
			}
			stack[len(stack)-1].sawElse = true
		case "endif":
			if len(stack) == 0 {
				document.Diagnostics = append(document.Diagnostics, ast.Diagnostic{
					URI:      document.URI,
					Range:    keywordRange,
					Severity: ast.SeverityError,
					Source:   "opsiscript-lsp",
					Message:  "endif without matching if",
				})
				continue
			}
			stack = stack[:len(stack)-1]
		}
	}

	flushOpenIfs(len(document.Lines))
}

func validateConditionalExpression(document *ast.Document, lineNumber int, keywordRange ast.Range, keyword, conditionText string) {
	condition := strings.TrimSpace(stripInlineComment(conditionText))
	if condition == "" {
		document.Diagnostics = append(document.Diagnostics, ast.Diagnostic{
			URI:      document.URI,
			Range:    keywordRange,
			Severity: ast.SeverityError,
			Source:   "opsiscript-lsp",
			Message:  fmt.Sprintf("missing condition after %s", keyword),
		})
		return
	}

	state := scanConditionDelimiters(condition)
	if state.quote != 0 {
		document.Diagnostics = append(document.Diagnostics, ast.Diagnostic{
			URI:      document.URI,
			Range:    ast.NewRange(lineNumber, keywordRange.End.Character, keywordRange.End.Character+len(conditionText)),
			Severity: ast.SeverityError,
			Source:   "opsiscript-lsp",
			Message:  fmt.Sprintf("unterminated string in %s condition", keyword),
		})
		return
	}

	if state.firstBadChar >= 0 {
		offset := keywordRange.End.Character + strings.Index(conditionText, condition)
		document.Diagnostics = append(document.Diagnostics, ast.Diagnostic{
			URI:      document.URI,
			Range:    ast.NewRange(lineNumber, offset+state.firstBadChar, offset+state.firstBadChar+1),
			Severity: ast.SeverityError,
			Source:   "opsiscript-lsp",
			Message:  fmt.Sprintf("unbalanced parentheses in %s condition", keyword),
		})
		return
	}

	if state.parentheses != 0 {
		document.Diagnostics = append(document.Diagnostics, ast.Diagnostic{
			URI:      document.URI,
			Range:    ast.NewRange(lineNumber, keywordRange.End.Character, keywordRange.End.Character+len(conditionText)),
			Severity: ast.SeverityError,
			Source:   "opsiscript-lsp",
			Message:  fmt.Sprintf("unbalanced parentheses in %s condition", keyword),
		})
		return
	}

	if logicalOperatorSuffixPattern.MatchString(condition) {
		document.Diagnostics = append(document.Diagnostics, ast.Diagnostic{
			URI:      document.URI,
			Range:    ast.NewRange(lineNumber, keywordRange.End.Character, keywordRange.End.Character+len(conditionText)),
			Severity: ast.SeverityError,
			Source:   "opsiscript-lsp",
			Message:  fmt.Sprintf("%s condition cannot end with logical operator", keyword),
		})
		return
	}

	comparator, comparatorOffset, ok := findMalformedComparator(condition)
	if !ok {
		return
	}

	offset := keywordRange.End.Character + strings.Index(conditionText, condition)
	document.Diagnostics = append(document.Diagnostics, ast.Diagnostic{
		URI:      document.URI,
		Range:    ast.NewRange(lineNumber, offset+comparatorOffset, offset+comparatorOffset+len(comparator)),
		Severity: ast.SeverityError,
		Source:   "opsiscript-lsp",
		Message:  fmt.Sprintf("malformed comparison in %s condition", keyword),
	})
}

func stripInlineComment(text string) string {
	var quote rune
	for index := 0; index < len(text); index++ {
		char := rune(text[index])
		if quote != 0 {
			if char == quote {
				quote = 0
			}
			continue
		}

		switch char {
		case '"', '\'':
			quote = char
		case ';':
			return text[:index]
		case '/':
			if index+1 < len(text) && text[index+1] == '/' {
				return text[:index]
			}
		}
	}

	return text
}

func scanConditionDelimiters(text string) delimiterState {
	state := delimiterState{firstBadChar: -1}
	for index := 0; index < len(text); index++ {
		char := rune(text[index])
		if state.quote != 0 {
			if char == state.quote {
				state.quote = 0
			}
			continue
		}

		switch char {
		case '"', '\'':
			state.quote = char
		case '(':
			state.parentheses++
		case ')':
			state.parentheses--
			if state.parentheses < 0 && state.firstBadChar < 0 {
				state.firstBadChar = index
			}
		}
	}
	return state
}

func findMalformedComparator(condition string) (string, int, bool) {
	masked := maskQuotedCondition(condition)
	for index := 0; index < len(masked); {
		comparator, width := comparatorAt(masked, index)
		if comparator == "" {
			index++
			continue
		}

		left := previousConditionToken(masked[:index])
		right := nextConditionToken(masked[index+width:])
		if !validComparatorOperand(left, false) || !validComparatorOperand(right, true) {
			return comparator, index, true
		}
		index += width
	}

	return "", 0, false
}

func maskQuotedCondition(text string) string {
	masked := []byte(text)
	var quote byte
	for index := 0; index < len(masked); index++ {
		char := masked[index]
		if quote != 0 {
			if char == quote {
				quote = 0
			}
			masked[index] = 'q'
			continue
		}
		if char == '"' || char == '\'' {
			quote = char
			masked[index] = 'q'
		}
	}
	return string(masked)
}

func comparatorAt(text string, index int) (string, int) {
	switch {
	case strings.HasPrefix(text[index:], "<>"):
		return "<>", 2
	case strings.HasPrefix(text[index:], "<="):
		return "<=", 2
	case strings.HasPrefix(text[index:], ">="):
		return ">=", 2
	case strings.HasPrefix(text[index:], "="):
		return "=", 1
	case strings.HasPrefix(text[index:], "<"):
		return "<", 1
	case strings.HasPrefix(text[index:], ">"):
		return ">", 1
	default:
		return "", 0
	}
}

func previousConditionToken(text string) string {
	end := len(text) - 1
	for end >= 0 && isConditionSpace(text[end]) {
		end--
	}
	if end < 0 {
		return ""
	}

	start := end
	for start >= 0 && !isConditionSpace(text[start]) {
		start--
	}
	return strings.ToLower(text[start+1 : end+1])
}

func nextConditionToken(text string) string {
	start := 0
	for start < len(text) && isConditionSpace(text[start]) {
		start++
	}
	if start >= len(text) {
		return ""
	}

	end := start
	for end < len(text) && !isConditionSpace(text[end]) {
		end++
	}
	return strings.ToLower(text[start:end])
}

func validComparatorOperand(token string, rightSide bool) bool {
	if token == "" {
		return false
	}

	switch token {
	case "and", "or", "=", "<>", "<=", ">=", "<", ">", "(":
		return false
	case ")":
		return !rightSide
	}

	return true
}

func isConditionSpace(char byte) bool {
	return char == ' ' || char == '\t' || char == '\r' || char == '\n'
}
