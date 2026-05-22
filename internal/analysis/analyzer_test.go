package analysis_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChristianBoehm/go-opsiscript-lsp/internal/analysis"
	"github.com/ChristianBoehm/go-opsiscript-lsp/internal/parser"
	"github.com/ChristianBoehm/go-opsiscript-lsp/internal/symbols"
)

func TestAnalyzeFindsDuplicatesAndMissingSections(t *testing.T) {
	text := strings.Join([]string{
		"[Actions]",
		"[Actions]",
		"DefVar $ProductId$",
		"DefVar $ProductId$",
		"Files_MissingPayload",
	}, "\n")

	document := parser.Parse("file:///dup.opsiscript", text)
	index := symbols.BuildIndex(document)
	analysis.Analyze(document, index)

	messages := make([]string, 0, len(document.Diagnostics))
	for _, diagnostic := range document.Diagnostics {
		messages = append(messages, diagnostic.Message)
	}

	assertContains(t, messages, `duplicate section name "Actions"`)
	assertContains(t, messages, `duplicate variable declaration "$ProductId$"`)
	assertContains(t, messages, `unknown section call "Files_MissingPayload"`)
}

func TestAnalyzeResolvesIncludedSections(t *testing.T) {
	setup := readFixture(t, "setup.opsiscript")
	sections := readFixture(t, "sections.opsiinc")
	declarations := readFixture(t, "declarations.opsiinc")
	delinc := readFixture(t, "delinc.opsiinc")
	osdLib := readFixture(t, "osd-lib.opsiscript")

	setupDoc := parser.Parse("file:///setup.opsiscript", setup)
	sectionsDoc := parser.Parse("file:///sections.opsiinc", sections)
	declarationsDoc := parser.Parse("file:///declarations.opsiinc", declarations)
	delincDoc := parser.Parse("file:///delinc.opsiinc", delinc)
	osdLibDoc := parser.Parse("file:///osd-lib.opsiscript", osdLib)

	index := symbols.BuildIndexDocuments(setupDoc, sectionsDoc, declarationsDoc, delincDoc, osdLibDoc)
	analysis.Analyze(setupDoc, index)

	for _, diagnostic := range setupDoc.Diagnostics {
		if diagnostic.Message == `unknown section call "Winbatch_install_1"` {
			t.Fatalf("unexpected unknown section diagnostic: %v", setupDoc.Diagnostics)
		}
	}
}

func TestAnalyzeConditionalStructure(t *testing.T) {
	text := strings.Join([]string{
		"[Actions]",
		"if $A$ = \"1\"",
		"  comment \"ok\"",
		"else",
		"  comment \"still ok\"",
		"endif",
		"elseif $A$ = \"2\"",
		"else",
		"endif",
		"if $B$ = \"1\"",
		"else",
		"elseif $B$ = \"2\"",
		"if $C$ = \"1\"",
		"[Sub_Test]",
		"endif",
	}, "\n")

	document := parser.Parse("file:///conditionals.opsiscript", text)
	index := symbols.BuildIndex(document)
	analysis.Analyze(document, index)

	messages := make([]string, 0, len(document.Diagnostics))
	for _, diagnostic := range document.Diagnostics {
		messages = append(messages, diagnostic.Message)
	}

	assertContains(t, messages, "elseif without matching if")
	assertContains(t, messages, "else without matching if")
	assertContains(t, messages, "endif without matching if")
	assertContains(t, messages, "elseif after else in the same if block")
	assertContains(t, messages, "missing endif for if block")
}

func TestAnalyzeConditionalExpressions(t *testing.T) {
	text := strings.Join([]string{
		"[Actions]",
		"if",
		"elseif",
		"if ($A$ = \"1\"",
		"if $A$ =",
		"if = \"1\"",
		"if $A$ = \"1\" and",
		"if StringContains($A$, \"x\")",
		"if ($A$ = \"1\") or ($B$ = \"2\") ; valid trailing comment",
	}, "\n")

	document := parser.Parse("file:///conditional-expressions.opsiscript", text)
	index := symbols.BuildIndex(document)
	analysis.Analyze(document, index)

	messages := make([]string, 0, len(document.Diagnostics))
	for _, diagnostic := range document.Diagnostics {
		messages = append(messages, diagnostic.Message)
	}

	assertContains(t, messages, "missing condition after if")
	assertContains(t, messages, "missing condition after elseif")
	assertContains(t, messages, "unbalanced parentheses in if condition")
	assertContains(t, messages, "malformed comparison in if condition")
	assertContains(t, messages, "if condition cannot end with logical operator")
}

func TestAnalyzeValidConditionalExpressions(t *testing.T) {
	text := strings.Join([]string{
		"[Actions]",
		"if StringContains($A$, \"x\")",
		"endif",
		"if ($A$ = \"1\") or ($B$ = \"2\") ; valid trailing comment",
		"endif",
	}, "\n")

	document := parser.Parse("file:///valid-conditional-expressions.opsiscript", text)
	index := symbols.BuildIndex(document)
	analysis.Analyze(document, index)

	for _, diagnostic := range document.Diagnostics {
		if strings.Contains(diagnostic.Message, "condition") || strings.Contains(diagnostic.Message, "comparison") {
			t.Fatalf("unexpected conditional expression diagnostic: %v", document.Diagnostics)
		}
	}
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("expected %q in diagnostics: %v", want, values)
}

func readFixture(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("..", "..", "..", "vscode-opsiscript-syntax", "examples", name)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}

	return string(content)
}
