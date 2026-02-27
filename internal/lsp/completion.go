package lsp

import (
	"strings"

	"github.com/fjl/geas/internal/ast"
	"github.com/fjl/geas/internal/evm"
)

// computeCompletion produces completion items at the given position.
func computeCompletion(doc *document, params CompletionParams) []CompletionItem {
	pos := params.Position

	// Get trigger character context.
	trigger := ""
	if params.Context != nil {
		trigger = params.Context.TriggerCharacter
	}

	// If no explicit trigger, figure out context from the line text.
	if trigger == "" {
		trigger = triggerFromLine(doc.Content, pos)
	}

	switch trigger {
	case "@":
		return completeLabelRefs(doc)
	case "%":
		return completeInstrMacros(doc)
	case "#":
		return completeDirectives(doc, pos)
	case ".":
		return completeBuiltins()
	default:
		return completeStatementPosition(doc)
	}
}

// triggerFromLine looks at the character before the cursor to determine context.
func triggerFromLine(content string, pos Position) string {
	lines := strings.SplitAfter(content, "\n")
	if pos.Line >= len(lines) {
		return ""
	}
	line := lines[pos.Line]
	if pos.Character <= 0 || pos.Character > len(line) {
		return ""
	}
	ch := line[pos.Character-1]
	switch ch {
	case '@', '%', '#', '.':
		return string(ch)
	}
	return ""
}

// completeStatementPosition returns opcodes and common identifiers.
func completeStatementPosition(doc *document) []CompletionItem {
	is := lookupFork(doc)
	ops := is.AllOps()

	items := make([]CompletionItem, 0, len(ops))
	for _, op := range ops {
		if op.Push && op.Name != "PUSH" && op.Name != "PUSH0" {
			continue // skip PUSH1-PUSH32, offer just PUSH
		}
		detail := ""
		in := op.StackIn(0)
		out := op.StackOut(0)
		if len(in) > 0 || len(out) > 0 {
			detail = "[" + strings.Join(in, ", ") + "] → [" + strings.Join(out, ", ") + "]"
		}
		items = append(items, CompletionItem{
			Label:  strings.ToLower(op.Name),
			Kind:   CompletionKindKeyword,
			Detail: detail,
		})
	}

	// Add generic PUSH.
	items = append(items, CompletionItem{
		Label:  "push",
		Kind:   CompletionKindKeyword,
		Detail: "Push value onto stack",
	})

	return items
}

// completeLabelRefs returns label names from the document.
func completeLabelRefs(doc *document) []CompletionItem {
	if doc.AST == nil {
		return nil
	}
	var items []CompletionItem
	for _, st := range doc.AST.Statements {
		if ld, ok := st.(*ast.LabelDef); ok {
			prefix := "@"
			if ld.Dotted {
				prefix = "@."
			}
			items = append(items, CompletionItem{
				Label:      prefix + ld.Ident,
				Kind:       CompletionKindVariable,
				InsertText: ld.Ident,
			})
		}
	}
	return items
}

// completeInstrMacros returns instruction macro names.
func completeInstrMacros(doc *document) []CompletionItem {
	if doc.AST == nil {
		return nil
	}
	var items []CompletionItem
	for _, st := range doc.AST.Statements {
		if def, ok := st.(*ast.InstructionMacroDef); ok {
			label := "%" + def.Ident
			insert := def.Ident
			if len(def.Params) > 0 {
				label += "(" + strings.Join(def.Params, ", ") + ")"
			}
			items = append(items, CompletionItem{
				Label:      label,
				Kind:       CompletionKindFunction,
				InsertText: insert,
			})
		}
	}
	return items
}

// completeDirectives returns directive completions.
func completeDirectives(doc *document, pos Position) []CompletionItem {
	// Check if this is #pragma target context.
	lines := strings.SplitAfter(doc.Content, "\n")
	if pos.Line < len(lines) {
		line := strings.TrimSpace(lines[pos.Line])
		if strings.HasPrefix(line, "#pragma") && strings.Contains(line, "target") {
			return completeForkNames()
		}
	}

	return []CompletionItem{
		{Label: "#define", Kind: CompletionKindKeyword, Detail: "Define a macro", InsertText: "define"},
		{Label: "#include", Kind: CompletionKindKeyword, Detail: "Include a file", InsertText: "include"},
		{Label: "#assemble", Kind: CompletionKindKeyword, Detail: "Assemble a file (deprecated)", InsertText: "assemble"},
		{Label: "#pragma", Kind: CompletionKindKeyword, Detail: "Set compiler option", InsertText: "pragma"},
		{Label: "#bytes", Kind: CompletionKindKeyword, Detail: "Insert raw bytes", InsertText: "bytes"},
	}
}

// completeBuiltins returns builtin macro completions.
func completeBuiltins() []CompletionItem {
	return []CompletionItem{
		{Label: ".len", Kind: CompletionKindFunction, Detail: "Byte length", InsertText: "len"},
		{Label: ".abs", Kind: CompletionKindFunction, Detail: "Absolute value", InsertText: "abs"},
		{Label: ".intbits", Kind: CompletionKindFunction, Detail: "Bit length", InsertText: "intbits"},
		{Label: ".selector", Kind: CompletionKindFunction, Detail: "ABI function selector", InsertText: "selector"},
		{Label: ".keccak256", Kind: CompletionKindFunction, Detail: "Keccak-256 hash", InsertText: "keccak256"},
		{Label: ".sha256", Kind: CompletionKindFunction, Detail: "SHA-256 hash", InsertText: "sha256"},
		{Label: ".address", Kind: CompletionKindFunction, Detail: "Checksummed address", InsertText: "address"},
		{Label: ".assemble", Kind: CompletionKindFunction, Detail: "Assemble file as value", InsertText: "assemble"},
	}
}

// completeForkNames returns all fork names as completion items.
func completeForkNames() []CompletionItem {
	forks := evm.AllForks()
	items := make([]CompletionItem, len(forks))
	for i, f := range forks {
		items[i] = CompletionItem{
			Label: f,
			Kind:  CompletionKindEnum,
		}
	}
	return items
}
