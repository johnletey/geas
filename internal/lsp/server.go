package lsp

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fjl/geas/asm"
	"github.com/fjl/geas/internal/ast"
	"github.com/fjl/geas/internal/evm"
)

// Server is the geas LSP server.
type Server struct {
	transport *transport
	store     *documentStore
	rootPath  string
	logger    *log.Logger
	shutdown  bool
}

// NewServer creates a new LSP server.
func NewServer(r io.Reader, w io.Writer) *Server {
	return &Server{
		transport: newTransport(r, w),
		store:     newDocumentStore(),
		logger:    log.New(os.Stderr, "geas-lsp: ", log.LstdFlags),
	}
}

// Run starts the server main loop.
func (s *Server) Run() error {
	for {
		msg, err := s.transport.readMessage()
		if err != nil {
			if s.shutdown {
				return nil
			}
			return fmt.Errorf("read error: %w", err)
		}
		if err := s.handleMessage(msg); err != nil {
			s.logger.Printf("handle error: %v", err)
		}
	}
}

func (s *Server) handleMessage(msg *jsonrpcMessage) error {
	switch msg.Method {
	// Lifecycle
	case "initialize":
		return s.handleInitialize(msg)
	case "initialized":
		return nil
	case "shutdown":
		s.shutdown = true
		return s.transport.sendResult(msg.ID, nil)
	case "exit":
		if s.shutdown {
			os.Exit(0)
		}
		os.Exit(1)
		return nil

	// Document sync
	case "textDocument/didOpen":
		return s.handleDidOpen(msg)
	case "textDocument/didChange":
		return s.handleDidChange(msg)
	case "textDocument/didClose":
		return s.handleDidClose(msg)
	case "textDocument/didSave":
		return nil

	// Language features
	case "textDocument/hover":
		return s.handleHover(msg)
	case "textDocument/definition":
		return s.handleDefinition(msg)
	case "textDocument/completion":
		return s.handleCompletion(msg)

	default:
		// Unknown method: send method-not-found for requests (have ID), ignore notifications.
		if msg.ID != nil {
			return s.transport.sendError(msg.ID, -32601, "method not found: "+msg.Method)
		}
		return nil
	}
}

func (s *Server) handleInitialize(msg *jsonrpcMessage) error {
	var params InitializeParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.transport.sendError(msg.ID, -32602, err.Error())
	}
	s.rootPath = uriToPath(params.RootURI)

	result := InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: 1, // full sync
			HoverProvider:    true,
			DefinitionProvider: true,
			CompletionProvider: &CompletionOpts{
				TriggerCharacters: []string{"@", "%", "#", "."},
			},
		},
	}
	return s.transport.sendResult(msg.ID, result)
}

func (s *Server) handleDidOpen(msg *jsonrpcMessage) error {
	var params DidOpenTextDocumentParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return err
	}
	doc := s.store.open(params.TextDocument.URI, params.TextDocument.Version, params.TextDocument.Text)
	return s.publishDiagnostics(doc)
}

func (s *Server) handleDidChange(msg *jsonrpcMessage) error {
	var params DidChangeTextDocumentParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return err
	}
	if len(params.ContentChanges) == 0 {
		return nil
	}
	// Full sync: use the last content change.
	content := params.ContentChanges[len(params.ContentChanges)-1].Text
	doc := s.store.change(params.TextDocument.URI, params.TextDocument.Version, content)
	if doc == nil {
		return nil
	}
	return s.publishDiagnostics(doc)
}

func (s *Server) handleDidClose(msg *jsonrpcMessage) error {
	var params DidCloseTextDocumentParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return err
	}
	// Clear diagnostics before closing.
	s.transport.sendNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
		URI:         params.TextDocument.URI,
		Diagnostics: []Diagnostic{},
	})
	s.store.close(params.TextDocument.URI)
	return nil
}

// publishDiagnostics runs the compiler and publishes diagnostics for a document.
func (s *Server) publishDiagnostics(doc *document) error {
	var diags []Diagnostic

	// Add parse errors.
	for _, pe := range doc.Errors {
		sev := SeverityError
		if pe.IsWarning() {
			sev = SeverityWarning
		}
		pos := pe.Position()
		diags = append(diags, Diagnostic{
			Range:    posToRange(pos),
			Severity: sev,
			Message:  pe.Unwrap().Error(),
		})
	}

	// Run compiler to get additional errors/warnings.
	if doc.AST != nil {
		c := asm.New(nil)
		if s.rootPath != "" {
			dir := filepath.Dir(doc.Path)
			c.SetFilesystem(s.store.overlayFS(dir))
		}
		c.CompileString(doc.Content)
		for _, err := range c.ErrorsAndWarnings() {
			sev := SeverityError
			if asm.IsWarning(err) {
				sev = SeverityWarning
			}
			pos := errorPosition(err)
			msg := errorMessage(err)
			diags = append(diags, Diagnostic{
				Range:    posToRange(pos),
				Severity: sev,
				Message:  msg,
			})
		}
	}

	if diags == nil {
		diags = []Diagnostic{}
	}
	return s.transport.sendNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
		URI:         doc.URI,
		Diagnostics: diags,
	})
}

// posToRange converts an ast.Position to an LSP Range.
func posToRange(pos ast.Position) Range {
	line := pos.Line - 1 // LSP is 0-based
	if line < 0 {
		line = 0
	}
	col := pos.Col
	return Range{
		Start: Position{Line: line, Character: col},
		End:   Position{Line: line, Character: col},
	}
}

// errorPosition extracts a Position from an error if possible.
func errorPosition(err error) ast.Position {
	type posError interface {
		Position() ast.Position
	}
	if pe, ok := err.(posError); ok {
		return pe.Position()
	}
	return ast.Position{}
}

// errorMessage extracts the inner message from a compiler error.
func errorMessage(err error) string {
	msg := err.Error()
	// Strip the "file:line: " prefix that the compiler adds.
	if idx := strings.Index(msg, ": "); idx >= 0 {
		rest := msg[idx+2:]
		// If it starts with "warning: ", strip that too.
		rest = strings.TrimPrefix(rest, "warning: ")
		return rest
	}
	return msg
}

// --- Hover ---

func (s *Server) handleHover(msg *jsonrpcMessage) error {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.transport.sendError(msg.ID, -32602, err.Error())
	}

	doc := s.store.get(params.TextDocument.URI)
	if doc == nil {
		return s.transport.sendResult(msg.ID, nil)
	}

	result := computeHover(doc, params.Position)
	return s.transport.sendResult(msg.ID, result)
}

// --- Definition ---

func (s *Server) handleDefinition(msg *jsonrpcMessage) error {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.transport.sendError(msg.ID, -32602, err.Error())
	}

	doc := s.store.get(params.TextDocument.URI)
	if doc == nil {
		return s.transport.sendResult(msg.ID, nil)
	}

	result := computeDefinition(doc, params.Position)
	return s.transport.sendResult(msg.ID, result)
}

// --- Completion ---

func (s *Server) handleCompletion(msg *jsonrpcMessage) error {
	var params CompletionParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return s.transport.sendError(msg.ID, -32602, err.Error())
	}

	doc := s.store.get(params.TextDocument.URI)
	if doc == nil {
		return s.transport.sendResult(msg.ID, []CompletionItem{})
	}

	result := computeCompletion(doc, params)
	return s.transport.sendResult(msg.ID, result)
}

// --- Token finding ---

// tokenAtPosition re-lexes the document and finds the token at the given position.
func tokenAtPosition(content string, pos Position) (ast.Token, bool) {
	tokens := ast.LexAll([]byte(content))
	for _, tok := range tokens {
		tokLine := tok.Line - 1 // convert to 0-based
		if tokLine != pos.Line {
			continue
		}
		// Compute token range.
		startCol := tok.Col
		endCol := startCol + len(tok.Text)
		// Adjust for prefix characters stripped by the lexer.
		switch tok.Type {
		case ast.TokenDottedLabelRef:
			startCol -= 2 // @. prefix
		case ast.TokenLabelRef, ast.TokenVariableIdentifier, ast.TokenInstMacroIdent:
			startCol-- // @, $, % prefix
		case ast.TokenDottedIdentifier, ast.TokenDottedLabel:
			startCol-- // . prefix
		}
		if pos.Character >= startCol && pos.Character < endCol {
			return tok, true
		}
	}
	return ast.Token{}, false
}

// lookupFork determines the EVM fork configured by #pragma target in the document.
func lookupFork(doc *document) *evm.InstructionSet {
	if doc.AST == nil {
		return evm.FindInstructionSet(evm.LatestFork)
	}
	for _, st := range doc.AST.Statements {
		if p, ok := st.(*ast.Pragma); ok && p.Option == "target" {
			if is := evm.FindInstructionSet(p.Value); is != nil {
				return is
			}
		}
	}
	return evm.FindInstructionSet(evm.LatestFork)
}
