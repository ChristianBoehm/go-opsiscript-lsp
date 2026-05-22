package ast

type Position struct {
	Line      int
	Character int
}

type Range struct {
	Start Position
	End   Position
}

func NewRange(line, startChar, endChar int) Range {
	return Range{
		Start: Position{Line: line, Character: startChar},
		End:   Position{Line: line, Character: endChar},
	}
}

func (r Range) Contains(pos Position) bool {
	if pos.Line < r.Start.Line || pos.Line > r.End.Line {
		return false
	}
	if pos.Line == r.Start.Line && pos.Character < r.Start.Character {
		return false
	}
	if pos.Line == r.End.Line && pos.Character > r.End.Character {
		return false
	}
	return true
}

type DiagnosticSeverity int

const (
	SeverityError   DiagnosticSeverity = 1
	SeverityWarning DiagnosticSeverity = 2
	SeverityInfo    DiagnosticSeverity = 3
	SeverityHint    DiagnosticSeverity = 4
)

type Diagnostic struct {
	URI      string
	Range    Range
	Severity DiagnosticSeverity
	Source   string
	Message  string
}

type Section struct {
	URI            string
	Name           string
	NormalizedName string
	Prefix         string
	Kind           string
	Range          Range
	NameRange      Range
}

type VariableDecl struct {
	URI            string
	Name           string
	NormalizedName string
	Kind           string
	Scope          string
	Range          Range
	NameRange      Range
}

type FunctionDecl struct {
	URI            string
	Name           string
	NormalizedName string
	ReturnType     string
	Range          Range
	NameRange      Range
}

type ReferenceKind string

const (
	ReferenceSection  ReferenceKind = "section"
	ReferenceVariable ReferenceKind = "variable"
	ReferenceFunction ReferenceKind = "function"
	ReferenceCommand  ReferenceKind = "command"
	ReferenceConstant ReferenceKind = "constant"
)

type Reference struct {
	URI            string
	Name           string
	NormalizedName string
	Kind           ReferenceKind
	Scope          string
	CallStyle      string
	Modifiers      []string
	Range          Range
}

type Include struct {
	Command string
	Target  string
	Range   Range
}

type Document struct {
	URI         string
	Text        string
	Lines       []string
	Includes    []Include
	Sections    []Section
	Variables   []VariableDecl
	Functions   []FunctionDecl
	References  []Reference
	Diagnostics []Diagnostic
}
