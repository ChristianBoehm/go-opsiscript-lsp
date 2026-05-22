package lsp

import (
	"io"
	"log"
	"strings"
	"testing"
)

func TestCompletionAddsConditionalSnippetsInSections(t *testing.T) {
	server := NewServer(strings.NewReader(""), io.Discard, log.New(io.Discard, "", 0))
	server.updateDocument("file:///test.opsiscript", strings.Join([]string{
		"DefVar $TopLevel$",
		"[Actions]",
		"",
	}, "\n"), 1)

	completions := server.completion(TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///test.opsiscript"},
		Position:     Position{Line: 2, Character: 0},
	})

	ifItem := findCompletion(completions.Items, "if")
	if ifItem == nil || ifItem.InsertTextFormat != 2 || !strings.Contains(ifItem.InsertText, "endif") {
		t.Fatalf("expected if snippet completion, got %#v", ifItem)
	}

	ifElseItem := findCompletion(completions.Items, "ifelse")
	if ifElseItem == nil || !strings.Contains(ifElseItem.InsertText, "else") {
		t.Fatalf("expected ifelse snippet completion, got %#v", ifElseItem)
	}

	if findCompletion(completions.Items, "else") != nil {
		t.Fatalf("did not expect else snippet outside an open if block")
	}
	if findCompletion(completions.Items, "endif") != nil {
		t.Fatalf("did not expect endif snippet outside an open if block")
	}
}

func TestCompletionAddsBranchSnippetsInsideOpenIf(t *testing.T) {
	server := NewServer(strings.NewReader(""), io.Discard, log.New(io.Discard, "", 0))
	server.updateDocument("file:///test.opsiscript", strings.Join([]string{
		"[Actions]",
		"if $Flag$ = \"1\"",
		"  ",
	}, "\n"), 1)

	completions := server.completion(TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///test.opsiscript"},
		Position:     Position{Line: 2, Character: 2},
	})

	for _, label := range []string{"if", "ifelse", "elseif", "else", "endif"} {
		if findCompletion(completions.Items, label) == nil {
			t.Fatalf("expected %s snippet in open if block", label)
		}
	}
}

func TestCompletionHidesElseAfterElseBranch(t *testing.T) {
	server := NewServer(strings.NewReader(""), io.Discard, log.New(io.Discard, "", 0))
	server.updateDocument("file:///test.opsiscript", strings.Join([]string{
		"[Actions]",
		"if $Flag$ = \"1\"",
		"else",
		"  ",
	}, "\n"), 1)

	completions := server.completion(TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///test.opsiscript"},
		Position:     Position{Line: 3, Character: 2},
	})

	if findCompletion(completions.Items, "else") != nil {
		t.Fatalf("did not expect duplicate else snippet after else branch")
	}
	if findCompletion(completions.Items, "endif") == nil {
		t.Fatalf("expected endif snippet after else branch")
	}
}

func findCompletion(items []CompletionItem, label string) *CompletionItem {
	for _, item := range items {
		if item.Label == label {
			value := item
			return &value
		}
	}
	return nil
}
