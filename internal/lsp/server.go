package lsp

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"regexp"
	"sort"
	"strings"

	"github.com/ChristianBoehm/go-opsiscript-lsp/internal/analysis"
	"github.com/ChristianBoehm/go-opsiscript-lsp/internal/ast"
	"github.com/ChristianBoehm/go-opsiscript-lsp/internal/symbols"
)

var conditionalKeywordPattern = regexp.MustCompile(`(?i)^\s*(if|elseif|else|endif)\b`)

type documentState struct {
	Document  *ast.Document
	Documents []*ast.Document
	Index     *symbols.Index
	Version   int
}

type Server struct {
	conn      *connection
	logger    *log.Logger
	documents map[string]*documentState
	shutdown  bool
}

func NewServer(reader io.Reader, writer io.Writer, logger *log.Logger) *Server {
	return &Server{
		conn:      newConnection(reader, writer),
		logger:    logger,
		documents: map[string]*documentState{},
	}
}

func (s *Server) Run() error {
	for {
		payload, err := s.conn.read()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		var message envelope
		if err := json.Unmarshal(payload, &message); err != nil {
			return err
		}

		if message.Method == "" {
			continue
		}

		if len(message.ID) == 0 {
			if err := s.handleNotification(message.Method, message.Params); err != nil {
				s.logger.Printf("notification %s failed: %v", message.Method, err)
			}
			if message.Method == "exit" {
				return nil
			}
			continue
		}

		result, responseErr := s.handleRequest(message.Method, message.Params)
		response := envelope{
			JSONRPC: "2.0",
			ID:      message.ID,
		}
		if responseErr != nil {
			response.Error = responseErr
		} else {
			if result == nil {
				response.Result = json.RawMessage("null")
			} else {
				response.Result = result
			}
		}

		if err := s.conn.write(response); err != nil {
			return err
		}
	}
}

func (s *Server) handleRequest(method string, params json.RawMessage) (any, *responseError) {
	switch method {
	case "initialize":
		return InitializeResult{
			Capabilities: ServerCapabilities{
				TextDocumentSync:       textDocumentSyncFull,
				HoverProvider:          true,
				CompletionProvider:     &CompletionOptions{},
				DocumentSymbolProvider: true,
				DefinitionProvider:     true,
				ReferencesProvider:     true,
			},
			ServerInfo: ServerInfo{
				Name:    "go-opsiscript-lsp",
				Version: "0.1.0",
			},
		}, nil
	case "shutdown":
		s.shutdown = true
		return nil, nil
	case "textDocument/hover":
		var request TextDocumentPositionParams
		if err := json.Unmarshal(params, &request); err != nil {
			return nil, invalidParams(err)
		}
		return s.hover(request), nil
	case "textDocument/completion":
		var request TextDocumentPositionParams
		if err := json.Unmarshal(params, &request); err != nil {
			return nil, invalidParams(err)
		}
		return s.completion(request), nil
	case "textDocument/documentSymbol":
		var request struct {
			TextDocument TextDocumentIdentifier `json:"textDocument"`
		}
		if err := json.Unmarshal(params, &request); err != nil {
			return nil, invalidParams(err)
		}
		return s.documentSymbols(request.TextDocument.URI), nil
	case "textDocument/definition":
		var request TextDocumentPositionParams
		if err := json.Unmarshal(params, &request); err != nil {
			return nil, invalidParams(err)
		}
		return s.definition(request), nil
	case "textDocument/references":
		var request ReferenceParams
		if err := json.Unmarshal(params, &request); err != nil {
			return nil, invalidParams(err)
		}
		return s.references(request), nil
	default:
		return nil, &responseError{
			Code:    -32601,
			Message: fmt.Sprintf("method %q not implemented", method),
		}
	}
}

func (s *Server) handleNotification(method string, params json.RawMessage) error {
	switch method {
	case "initialized":
		return nil
	case "exit":
		return nil
	case "textDocument/didOpen":
		var request DidOpenTextDocumentParams
		if err := json.Unmarshal(params, &request); err != nil {
			return err
		}
		s.updateDocument(request.TextDocument.URI, request.TextDocument.Text, request.TextDocument.Version)
		return s.publishDiagnostics(request.TextDocument.URI)
	case "textDocument/didChange":
		var request DidChangeTextDocumentParams
		if err := json.Unmarshal(params, &request); err != nil {
			return err
		}
		if len(request.ContentChanges) == 0 {
			return nil
		}
		text := request.ContentChanges[len(request.ContentChanges)-1].Text
		s.updateDocument(request.TextDocument.URI, text, request.TextDocument.Version)
		return s.publishDiagnostics(request.TextDocument.URI)
	case "textDocument/didClose":
		var request DidCloseTextDocumentParams
		if err := json.Unmarshal(params, &request); err != nil {
			return err
		}
		delete(s.documents, request.TextDocument.URI)
		return s.conn.write(envelope{
			JSONRPC: "2.0",
			Method:  "textDocument/publishDiagnostics",
			Params: mustMarshalRaw(PublishDiagnosticsParams{
				URI:         request.TextDocument.URI,
				Diagnostics: []Diagnostic{},
			}),
		})
	default:
		return nil
	}
}

func (s *Server) updateDocument(uri, text string, version int) {
	documents := loadWorkspaceDocuments(uri, text)
	document := documents[0]
	index := symbols.BuildIndexDocuments(documents...)
	analysis.Analyze(document, index)
	s.documents[uri] = &documentState{
		Document:  document,
		Documents: documents,
		Index:     index,
		Version:   version,
	}
}

func (s *Server) publishDiagnostics(uri string) error {
	state := s.documents[uri]
	if state == nil {
		return nil
	}

	diagnostics := make([]Diagnostic, 0, len(state.Document.Diagnostics))
	for _, diagnostic := range state.Document.Diagnostics {
		if diagnostic.URI != "" && diagnostic.URI != uri {
			continue
		}
		diagnostics = append(diagnostics, Diagnostic{
			Range:    toProtocolRange(diagnostic.Range),
			Severity: int(diagnostic.Severity),
			Source:   diagnostic.Source,
			Message:  diagnostic.Message,
		})
	}

	return s.conn.write(envelope{
		JSONRPC: "2.0",
		Method:  "textDocument/publishDiagnostics",
		Params: mustMarshalRaw(PublishDiagnosticsParams{
			URI:         uri,
			Version:     state.Version,
			Diagnostics: diagnostics,
		}),
	})
}

func (s *Server) hover(request TextDocumentPositionParams) *Hover {
	state := s.documents[request.TextDocument.URI]
	if state == nil {
		return nil
	}

	position := toASTPosition(request.Position)
	if variable := variableAt(state.Document, position); variable != nil {
		scope := "global scope"
		if variable.Scope != "" {
			scope = fmt.Sprintf("function `%s`", variable.Scope)
		}
		return &Hover{
			Contents: MarkupContent{
				Kind:  "markdown",
				Value: fmt.Sprintf("**Variable** `%s`\n\nDeclared as `%s` in %s.", variable.Name, variable.Kind, scope),
			},
			Range: refRangePtr(variable.NameRange),
		}
	}

	if section := sectionAt(state.Document, position); section != nil {
		return &Hover{
			Contents: MarkupContent{
				Kind:  "markdown",
				Value: fmt.Sprintf("**Section** `%s`\n\n%s.", section.Name, section.Kind),
			},
			Range: refRangePtr(section.NameRange),
		}
	}

	if function := functionAt(state.Document, position); function != nil {
		detail := "**Function**"
		if function.ReturnType != "" {
			detail = fmt.Sprintf("**Function** `%s` → `%s`", function.Name, function.ReturnType)
		} else {
			detail = fmt.Sprintf("**Function** `%s`", function.Name)
		}
		return &Hover{
			Contents: MarkupContent{
				Kind:  "markdown",
				Value: detail,
			},
			Range: refRangePtr(function.NameRange),
		}
	}

	if reference := referenceAt(state.Document, position); reference != nil {
		switch reference.Kind {
		case ast.ReferenceVariable:
			if variable := symbols.ResolveVariable(state.Index, reference.Name, reference.Scope); variable != nil {
				scope := "global scope"
				if variable.Scope != "" {
					scope = fmt.Sprintf("function `%s`", variable.Scope)
				}
				return &Hover{
					Contents: MarkupContent{
						Kind:  "markdown",
						Value: fmt.Sprintf("**Variable** `%s`\n\nDeclared as `%s` in %s.", variable.Name, variable.Kind, scope),
					},
					Range: refRangePtr(reference.Range),
				}
			}
		case ast.ReferenceSection:
			if section := symbols.ResolveSection(state.Index, reference.Name); section != nil {
				value := fmt.Sprintf("**Section** `%s`\n\n%s.", section.Name, section.Kind)
				if len(reference.Modifiers) > 0 {
					value += fmt.Sprintf("\n\nCall modifiers: `%s`.", strings.Join(reference.Modifiers, "`, `"))
				}
				return &Hover{
					Contents: MarkupContent{
						Kind:  "markdown",
						Value: value,
					},
					Range: refRangePtr(reference.Range),
				}
			}
		case ast.ReferenceFunction:
			if builtin, ok := symbols.LookupFunction(reference.Name); ok {
				return &Hover{
					Contents: MarkupContent{
						Kind:  "markdown",
						Value: fmt.Sprintf("**Builtin function** `%s`\n\n%s", builtin.Label, builtin.Documentation),
					},
					Range: refRangePtr(reference.Range),
				}
			}
			if function := symbols.ResolveFunction(state.Index, reference.Name); function != nil {
				value := fmt.Sprintf("**Function** `%s`", function.Name)
				if function.ReturnType != "" {
					value = fmt.Sprintf("**Function** `%s` → `%s`", function.Name, function.ReturnType)
				}
				return &Hover{
					Contents: MarkupContent{
						Kind:  "markdown",
						Value: value,
					},
					Range: refRangePtr(reference.Range),
				}
			}
		case ast.ReferenceCommand:
			if builtin, ok := symbols.LookupCommand(reference.Name); ok {
				return &Hover{
					Contents: MarkupContent{
						Kind:  "markdown",
						Value: fmt.Sprintf("**Builtin command** `%s`\n\n%s", builtin.Label, builtin.Documentation),
					},
					Range: refRangePtr(reference.Range),
				}
			}
		case ast.ReferenceConstant:
			if builtin, ok := symbols.LookupConstant(reference.Name); ok {
				return &Hover{
					Contents: MarkupContent{
						Kind:  "markdown",
						Value: fmt.Sprintf("**Builtin constant** `%s`\n\n%s", builtin.Label, builtin.Documentation),
					},
					Range: refRangePtr(reference.Range),
				}
			}
		}
	}

	return nil
}

func (s *Server) completion(request TextDocumentPositionParams) CompletionList {
	state := s.documents[request.TextDocument.URI]
	if state == nil {
		return CompletionList{IsIncomplete: false, Items: nil}
	}

	items := make([]CompletionItem, 0, 64)
	seen := map[string]struct{}{}

	addItem := func(label string, kind int, detail, documentation string) {
		key := strings.ToLower(label)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		item := CompletionItem{
			Label:  label,
			Kind:   kind,
			Detail: detail,
		}
		if documentation != "" {
			item.Documentation = &MarkupContent{Kind: "markdown", Value: documentation}
		}
		items = append(items, item)
	}
	addCompletionItem := func(item CompletionItem) {
		key := strings.ToLower(item.Label)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		items = append(items, item)
	}

	addConditionalSnippetCompletions(state.Document, request.Position, addCompletionItem)

	for _, name := range symbols.CommandNames() {
		builtin, _ := symbols.LookupCommand(name)
		addItem(name, 14, builtin.Detail, builtin.Documentation)
	}
	for _, name := range symbols.FunctionNames() {
		builtin, _ := symbols.LookupFunction(name)
		addItem(name, 3, builtin.Detail, builtin.Documentation)
	}
	for _, name := range symbols.ConstantNames() {
		builtin, _ := symbols.LookupConstant(name)
		addItem(name, 21, builtin.Detail, builtin.Documentation)
	}
	for _, document := range state.Documents {
		for _, section := range document.Sections {
			addItem(section.Name, 5, section.Kind, "")
		}
		for _, variable := range document.Variables {
			detail := variable.Kind
			if variable.Scope != "" {
				detail += " in " + variable.Scope
			}
			addItem(variable.Name, 6, detail, "")
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Label) < strings.ToLower(items[j].Label)
	})

	return CompletionList{
		IsIncomplete: false,
		Items:        items,
	}
}

func addConditionalSnippetCompletions(document *ast.Document, position Position, addItem func(CompletionItem)) {
	if !isScriptSectionPosition(document, position) {
		return
	}

	indent := lineIndentAtPosition(document, position)
	bodyIndent := indent + "  "

	addItem(CompletionItem{
		Label:            "if",
		Kind:             15,
		Detail:           "Conditional block snippet",
		Documentation:    snippetDocumentation("if", "Insert an if ... endif block."),
		InsertText:       "if ${1:condition}\n" + bodyIndent + "$0\n" + indent + "endif",
		InsertTextFormat: 2,
	})
	addItem(CompletionItem{
		Label:            "ifelse",
		Kind:             15,
		Detail:           "Conditional block snippet",
		Documentation:    snippetDocumentation("ifelse", "Insert an if ... else ... endif block."),
		InsertText:       "if ${1:condition}\n" + bodyIndent + "${2}\n" + indent + "else\n" + bodyIndent + "$0\n" + indent + "endif",
		InsertTextFormat: 2,
	})

	ifState := conditionalStateAt(document, position)
	if ifState.depth == 0 {
		return
	}

	addItem(CompletionItem{
		Label:            "elseif",
		Kind:             15,
		Detail:           "Conditional branch snippet",
		Documentation:    snippetDocumentation("elseif", "Insert an elseif branch inside the current if block."),
		InsertText:       "elseif ${1:condition}\n" + bodyIndent + "$0",
		InsertTextFormat: 2,
	})
	if !ifState.currentHasElse {
		addItem(CompletionItem{
			Label:            "else",
			Kind:             15,
			Detail:           "Conditional branch snippet",
			Documentation:    snippetDocumentation("else", "Insert an else branch inside the current if block."),
			InsertText:       "else\n" + bodyIndent + "$0",
			InsertTextFormat: 2,
		})
	}
	addItem(CompletionItem{
		Label:            "endif",
		Kind:             15,
		Detail:           "Conditional closing snippet",
		Documentation:    snippetDocumentation("endif", "Close the current if block."),
		InsertText:       "endif",
		InsertTextFormat: 2,
	})
}

func snippetDocumentation(label, description string) *MarkupContent {
	return &MarkupContent{
		Kind:  "markdown",
		Value: fmt.Sprintf("**Snippet** `%s`\n\n%s", label, description),
	}
}

func isScriptSectionPosition(document *ast.Document, position Position) bool {
	if position.Line < 0 || position.Line >= len(document.Lines) {
		return false
	}
	line := document.Lines[position.Line]
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "//") {
		return false
	}

	for index := len(document.Sections) - 1; index >= 0; index-- {
		section := document.Sections[index]
		if section.Range.Start.Line <= position.Line {
			return true
		}
	}

	return false
}

func lineIndentAtPosition(document *ast.Document, position Position) string {
	if position.Line < 0 || position.Line >= len(document.Lines) {
		return ""
	}
	line := document.Lines[position.Line]
	if position.Character >= 0 && position.Character < len(line) {
		line = line[:position.Character]
	}
	trimmed := strings.TrimLeft(line, " \t")
	return line[:len(line)-len(trimmed)]
}

type conditionalCompletionState struct {
	depth          int
	currentHasElse bool
}

func conditionalStateAt(document *ast.Document, position Position) conditionalCompletionState {
	if position.Line < 0 || position.Line >= len(document.Lines) {
		return conditionalCompletionState{}
	}

	sectionStart := 0
	for _, section := range document.Sections {
		if section.Range.Start.Line <= position.Line {
			sectionStart = section.Range.Start.Line + 1
			continue
		}
		break
	}

	stack := make([]bool, 0, 4)
	for lineNumber := sectionStart; lineNumber <= position.Line; lineNumber++ {
		lineText := document.Lines[lineNumber]
		if lineNumber == position.Line && position.Character >= 0 && position.Character < len(lineText) {
			lineText = lineText[:position.Character]
		}

		trimmed := strings.TrimSpace(lineText)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "[") {
			continue
		}

		matches := conditionalKeywordPattern.FindStringSubmatchIndex(lineText)
		if matches == nil {
			continue
		}

		switch strings.ToLower(lineText[matches[2]:matches[3]]) {
		case "if":
			stack = append(stack, false)
		case "elseif":
			if len(stack) == 0 {
				continue
			}
		case "else":
			if len(stack) == 0 {
				continue
			}
			stack[len(stack)-1] = true
		case "endif":
			if len(stack) == 0 {
				continue
			}
			stack = stack[:len(stack)-1]
		}
	}

	if len(stack) == 0 {
		return conditionalCompletionState{}
	}

	return conditionalCompletionState{
		depth:          len(stack),
		currentHasElse: stack[len(stack)-1],
	}
}

func (s *Server) documentSymbols(uri string) []DocumentSymbol {
	state := s.documents[uri]
	if state == nil {
		return nil
	}

	symbolsList := make([]DocumentSymbol, 0, len(state.Document.Sections)+len(state.Document.Functions)+len(state.Document.Variables))
	for _, section := range state.Document.Sections {
		symbolsList = append(symbolsList, DocumentSymbol{
			Name:           section.Name,
			Detail:         section.Kind,
			Kind:           3,
			Range:          toProtocolRange(section.Range),
			SelectionRange: toProtocolRange(section.NameRange),
		})
	}
	for _, function := range state.Document.Functions {
		detail := function.ReturnType
		symbolsList = append(symbolsList, DocumentSymbol{
			Name:           function.Name,
			Detail:         detail,
			Kind:           12,
			Range:          toProtocolRange(function.Range),
			SelectionRange: toProtocolRange(function.NameRange),
		})
	}
	for _, variable := range state.Document.Variables {
		detail := variable.Kind
		if variable.Scope != "" {
			detail += " in " + variable.Scope
		}
		symbolsList = append(symbolsList, DocumentSymbol{
			Name:           variable.Name,
			Detail:         detail,
			Kind:           13,
			Range:          toProtocolRange(variable.Range),
			SelectionRange: toProtocolRange(variable.NameRange),
		})
	}

	sort.Slice(symbolsList, func(i, j int) bool {
		if symbolsList[i].Range.Start.Line == symbolsList[j].Range.Start.Line {
			return symbolsList[i].Range.Start.Character < symbolsList[j].Range.Start.Character
		}
		return symbolsList[i].Range.Start.Line < symbolsList[j].Range.Start.Line
	})

	return symbolsList
}

func (s *Server) definition(request TextDocumentPositionParams) []Location {
	state := s.documents[request.TextDocument.URI]
	if state == nil {
		return nil
	}

	position := toASTPosition(request.Position)

	if variable := variableAt(state.Document, position); variable != nil {
		return []Location{{URI: variable.URI, Range: toProtocolRange(variable.NameRange)}}
	}
	if section := sectionAt(state.Document, position); section != nil {
		return []Location{{URI: section.URI, Range: toProtocolRange(section.NameRange)}}
	}
	if function := functionAt(state.Document, position); function != nil {
		return []Location{{URI: function.URI, Range: toProtocolRange(function.NameRange)}}
	}

	reference := referenceAt(state.Document, position)
	if reference == nil {
		return nil
	}

	switch reference.Kind {
	case ast.ReferenceVariable:
		if variable := symbols.ResolveVariable(state.Index, reference.Name, reference.Scope); variable != nil {
			return []Location{{URI: variable.URI, Range: toProtocolRange(variable.NameRange)}}
		}
	case ast.ReferenceSection:
		if section := symbols.ResolveSection(state.Index, reference.Name); section != nil {
			return []Location{{URI: section.URI, Range: toProtocolRange(section.NameRange)}}
		}
	case ast.ReferenceFunction:
		if function := symbols.ResolveFunction(state.Index, reference.Name); function != nil {
			return []Location{{URI: function.URI, Range: toProtocolRange(function.NameRange)}}
		}
	}

	return nil
}

func (s *Server) references(request ReferenceParams) []Location {
	state := s.documents[request.TextDocument.URI]
	if state == nil {
		return nil
	}

	position := toASTPosition(request.Position)
	var locations []Location

	if variable := variableAt(state.Document, position); variable != nil {
		return variableReferences(state.Documents, state.Index, *variable, request.Context.IncludeDeclaration)
	}
	if section := sectionAt(state.Document, position); section != nil {
		return sectionReferences(state.Documents, *section, request.Context.IncludeDeclaration)
	}
	if function := functionAt(state.Document, position); function != nil {
		return functionReferences(state.Documents, *function, request.Context.IncludeDeclaration)
	}

	reference := referenceAt(state.Document, position)
	if reference == nil {
		return nil
	}

	switch reference.Kind {
	case ast.ReferenceVariable:
		if variable := symbols.ResolveVariable(state.Index, reference.Name, reference.Scope); variable != nil {
			locations = variableReferences(state.Documents, state.Index, *variable, request.Context.IncludeDeclaration)
		}
	case ast.ReferenceSection:
		if section := symbols.ResolveSection(state.Index, reference.Name); section != nil {
			locations = sectionReferences(state.Documents, *section, request.Context.IncludeDeclaration)
		}
	case ast.ReferenceFunction:
		if function := symbols.ResolveFunction(state.Index, reference.Name); function != nil {
			locations = functionReferences(state.Documents, *function, request.Context.IncludeDeclaration)
		}
	}

	return locations
}

func variableReferences(documents []*ast.Document, index *symbols.Index, declaration ast.VariableDecl, includeDeclaration bool) []Location {
	locations := make([]Location, 0, 8)
	if includeDeclaration {
		locations = append(locations, Location{URI: declaration.URI, Range: toProtocolRange(declaration.NameRange)})
	}
	for _, document := range documents {
		for _, reference := range document.References {
			if reference.Kind != ast.ReferenceVariable || reference.NormalizedName != declaration.NormalizedName {
				continue
			}
			resolved := symbols.ResolveVariable(index, reference.Name, reference.Scope)
			if !sameVariable(resolved, declaration) {
				continue
			}
			locations = append(locations, Location{URI: reference.URI, Range: toProtocolRange(reference.Range)})
		}
	}
	return locations
}

func sectionReferences(documents []*ast.Document, declaration ast.Section, includeDeclaration bool) []Location {
	locations := make([]Location, 0, 8)
	if includeDeclaration {
		locations = append(locations, Location{URI: declaration.URI, Range: toProtocolRange(declaration.NameRange)})
	}
	for _, document := range documents {
		for _, reference := range document.References {
			if reference.Kind != ast.ReferenceSection || reference.NormalizedName != declaration.NormalizedName {
				continue
			}
			locations = append(locations, Location{URI: reference.URI, Range: toProtocolRange(reference.Range)})
		}
	}
	return locations
}

func functionReferences(documents []*ast.Document, declaration ast.FunctionDecl, includeDeclaration bool) []Location {
	locations := make([]Location, 0, 8)
	if includeDeclaration {
		locations = append(locations, Location{URI: declaration.URI, Range: toProtocolRange(declaration.NameRange)})
	}
	for _, document := range documents {
		for _, reference := range document.References {
			if reference.Kind != ast.ReferenceFunction || reference.NormalizedName != declaration.NormalizedName {
				continue
			}
			locations = append(locations, Location{URI: reference.URI, Range: toProtocolRange(reference.Range)})
		}
	}
	return locations
}

func sameVariable(candidate *ast.VariableDecl, declaration ast.VariableDecl) bool {
	if candidate == nil {
		return false
	}
	return candidate.URI == declaration.URI && candidate.NameRange == declaration.NameRange
}

func variableAt(document *ast.Document, position ast.Position) *ast.VariableDecl {
	for _, variable := range document.Variables {
		if variable.NameRange.Contains(position) {
			value := variable
			return &value
		}
	}
	return nil
}

func sectionAt(document *ast.Document, position ast.Position) *ast.Section {
	for _, section := range document.Sections {
		if section.NameRange.Contains(position) {
			value := section
			return &value
		}
	}
	return nil
}

func functionAt(document *ast.Document, position ast.Position) *ast.FunctionDecl {
	for _, function := range document.Functions {
		if function.NameRange.Contains(position) {
			value := function
			return &value
		}
	}
	return nil
}

func referenceAt(document *ast.Document, position ast.Position) *ast.Reference {
	for _, reference := range document.References {
		if reference.Range.Contains(position) {
			value := reference
			return &value
		}
	}
	return nil
}

func invalidParams(err error) *responseError {
	return &responseError{
		Code:    -32602,
		Message: err.Error(),
	}
}

func toASTPosition(position Position) ast.Position {
	return ast.Position{
		Line:      position.Line,
		Character: position.Character,
	}
}

func toProtocolRange(value ast.Range) Range {
	return Range{
		Start: Position{Line: value.Start.Line, Character: value.Start.Character},
		End:   Position{Line: value.End.Line, Character: value.End.Character},
	}
}

func refRangePtr(value ast.Range) *Range {
	protocolRange := toProtocolRange(value)
	return &protocolRange
}
