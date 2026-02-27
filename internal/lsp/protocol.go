package lsp

// LSP type definitions for the subset needed by geas-lsp.

// Position in a text document (0-based line and character).
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range in a text document.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a location inside a resource.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// Diagnostic represents a compiler error or warning.
type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"`
	Message  string `json:"message"`
}

// Diagnostic severity constants.
const (
	SeverityError       = 1
	SeverityWarning     = 2
	SeverityInformation = 3
	SeverityHint        = 4
)

// TextDocumentIdentifier identifies a text document.
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// TextDocumentItem is an item transferred on open.
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// VersionedTextDocumentIdentifier identifies a versioned text document.
type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

// TextDocumentContentChangeEvent describes a content change.
type TextDocumentContentChangeEvent struct {
	Text string `json:"text"` // full sync mode: entire new content
}

// TextDocumentPositionParams is sent for hover/definition/completion.
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// CompletionParams is sent for completion requests.
type CompletionParams struct {
	TextDocumentPositionParams
	Context *CompletionContext `json:"context,omitempty"`
}

// CompletionContext contains additional info about the completion trigger.
type CompletionContext struct {
	TriggerKind      int    `json:"triggerKind"`
	TriggerCharacter string `json:"triggerCharacter,omitempty"`
}

// CompletionItem represents a completion suggestion.
type CompletionItem struct {
	Label         string `json:"label"`
	Kind          int    `json:"kind,omitempty"`
	Detail        string `json:"detail,omitempty"`
	Documentation string `json:"documentation,omitempty"`
	InsertText    string `json:"insertText,omitempty"`
}

// Completion item kinds.
const (
	CompletionKindText     = 1
	CompletionKindFunction = 3
	CompletionKindField    = 5
	CompletionKindVariable = 6
	CompletionKindKeyword  = 14
	CompletionKindValue    = 12
	CompletionKindEnum     = 13
)

// Hover result.
type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

// MarkupContent represents hover content.
type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// --- Request/notification parameter types ---

type InitializeParams struct {
	RootURI string `json:"rootUri"`
}

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// --- Server capabilities ---

type ServerCapabilities struct {
	TextDocumentSync   int              `json:"textDocumentSync"`
	HoverProvider      bool             `json:"hoverProvider"`
	CompletionProvider *CompletionOpts  `json:"completionProvider,omitempty"`
	DefinitionProvider bool             `json:"definitionProvider"`
}

type CompletionOpts struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}
