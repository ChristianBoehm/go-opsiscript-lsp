package lexer_test

import (
	"testing"

	"github.com/ChristianBoehm/go-opsiscript-lsp/internal/lexer"
)

func TestLexAcceptsWindowsPathBeforeQuote(t *testing.T) {
	result := lexer.Lex(`ShowBitmap "%ScriptPath%\" + $ProductId$ + ".png" $ProductId$`)

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}
}

func TestLexReportsUnclosedQuotes(t *testing.T) {
	result := lexer.Lex(`Message "unterminated`)

	if len(result.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(result.Diagnostics))
	}
}
