package parser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChristianBoehm/go-opsiscript-lsp/internal/parser"
)

func TestParseSampleFixture(t *testing.T) {
	text := readFixture(t, "sample.opsiscript")
	document := parser.Parse("file:///sample.opsiscript", text)

	if len(document.Sections) != 5 {
		t.Fatalf("expected 5 sections, got %d", len(document.Sections))
	}

	if len(document.Variables) != 3 {
		t.Fatalf("expected 3 variable declarations, got %d", len(document.Variables))
	}

	foundSectionCall := false
	for _, reference := range document.References {
		if reference.Name == "Files_CopyExample" {
			foundSectionCall = true
			break
		}
	}

	if !foundSectionCall {
		t.Fatalf("expected section call reference for Files_CopyExample")
	}
}

func TestParseRealWorldFixtures(t *testing.T) {
	for _, name := range []string{"setup.opsiscript", "uninstall.opsiscript", "osd-lib.opsiscript", "declarations.opsiinc", "sections.opsiinc", "delinc.opsiinc"} {
		t.Run(name, func(t *testing.T) {
			text := readFixture(t, name)
			document := parser.Parse("file:///"+name, text)

			if len(document.Lines) == 0 {
				t.Fatalf("expected lines for %s", name)
			}

			if strings.Contains(name, "osd-lib") && len(document.Functions) == 0 {
				t.Fatalf("expected function declarations in %s", name)
			}
		})
	}
}

func TestParseSectionModifiersAndAdditionalSectionFamilies(t *testing.T) {
	text := strings.Join([]string{
		"[Actions]",
		"PatchHosts_Update /64Bit",
		"LDAPsearch_Query winst /64Bit",
		`set $result$ = getOutStreamFromSection("ShellInAnIcon_Query /timeoutseconds 20")`,
		"[PatchHosts_Update]",
		"[LDAPsearch_Query]",
		"[ShellInAnIcon_Query]",
	}, "\n")

	document := parser.Parse("file:///modifiers.opsiscript", text)

	if len(document.Sections) != 4 {
		t.Fatalf("expected 4 sections, got %d", len(document.Sections))
	}

	var patchHostsReference, ldapReference, shellReference bool
	for _, reference := range document.References {
		switch reference.Name {
		case "PatchHosts_Update":
			patchHostsReference = len(reference.Modifiers) == 1 && reference.Modifiers[0] == "/64Bit"
		case "LDAPsearch_Query":
			ldapReference = len(reference.Modifiers) == 2 && reference.Modifiers[0] == "winst" && reference.Modifiers[1] == "/64Bit"
		case "ShellInAnIcon_Query":
			shellReference = len(reference.Modifiers) == 1 && reference.Modifiers[0] == "/timeoutseconds 20"
		}
	}

	if !patchHostsReference {
		t.Fatalf("expected PatchHosts reference with /64Bit modifier")
	}
	if !ldapReference {
		t.Fatalf("expected LDAPsearch reference with winst and /64Bit modifiers")
	}
	if !shellReference {
		t.Fatalf("expected ShellInAnIcon reference with /timeoutseconds modifier")
	}
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
