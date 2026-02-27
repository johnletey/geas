package lsp

import (
	"fmt"
	"strings"

	"github.com/fjl/geas/internal/ast"
	"github.com/fjl/geas/internal/evm"
)

// computeHover computes hover information for the symbol at the given position.
func computeHover(doc *document, pos Position) *Hover {
	tok, ok := tokenAtPosition(doc.Content, pos)
	if !ok {
		return nil
	}

	var content string

	switch tok.Type {
	case ast.TokenIdentifier:
		content = hoverIdentifier(doc, tok)
	case ast.TokenDottedIdentifier:
		content = hoverDottedIdentifier(tok)
	case ast.TokenLabelRef, ast.TokenDottedLabelRef:
		content = hoverLabelRef(doc, tok)
	case ast.TokenLabel, ast.TokenDottedLabel:
		content = hoverLabel(tok)
	case ast.TokenInstMacroIdent:
		content = hoverInstrMacro(doc, tok)
	case ast.TokenDirective:
		content = hoverDirective(tok)
	case ast.TokenVariableIdentifier:
		content = fmt.Sprintf("**Parameter** `$%s`", tok.Text)
	}

	if content == "" {
		return nil
	}
	return &Hover{
		Contents: MarkupContent{Kind: "markdown", Value: content},
	}
}

func hoverIdentifier(doc *document, tok ast.Token) string {
	// Check if it's an opcode.
	is := lookupFork(doc)
	upper := strings.ToUpper(tok.Text)
	if op := is.OpByName(upper); op != nil {
		return formatOpHover(op)
	}

	// Check if it's an expression macro.
	if doc.AST != nil {
		if def, _ := doc.AST.LookupExprMacro(tok.Text); def != nil {
			return formatExprMacroHover(def)
		}
	}
	return ""
}

func hoverDottedIdentifier(tok ast.Token) string {
	if doc, ok := builtinMacroDocs[tok.Text]; ok {
		return doc
	}
	return fmt.Sprintf("**Builtin** `.%s`", tok.Text)
}

func hoverLabelRef(doc *document, tok ast.Token) string {
	prefix := "@"
	if tok.Type == ast.TokenDottedLabelRef {
		prefix = "@."
	}
	info := fmt.Sprintf("**Label reference** `%s%s`", prefix, tok.Text)

	if doc.AST != nil {
		lref := &ast.LabelRefExpr{Ident: tok.Text, Dotted: tok.Type == ast.TokenDottedLabelRef}
		if ldef, _ := doc.AST.LookupLabel(lref); ldef != nil {
			kind := "jumpdest"
			if ldef.Dotted {
				kind = "dotted (no JUMPDEST)"
			}
			info += fmt.Sprintf("\n\nDefined as %s label", kind)
		}
	}
	return info
}

func hoverLabel(tok ast.Token) string {
	kind := "Label"
	if tok.Type == ast.TokenDottedLabel {
		kind = "Dotted label"
	}
	return fmt.Sprintf("**%s** `%s:`", kind, tok.Text)
}

func hoverInstrMacro(doc *document, tok ast.Token) string {
	if doc.AST != nil {
		if def, _ := doc.AST.LookupInstrMacro(tok.Text); def != nil {
			return formatInstrMacroHover(def)
		}
	}
	return fmt.Sprintf("**Instruction macro** `%%%s`", tok.Text)
}

func hoverDirective(tok ast.Token) string {
	if doc, ok := directiveDocs[tok.Text]; ok {
		return doc
	}
	return ""
}

func formatOpHover(op *evm.Op) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "**%s** `0x%02x`\n\n", op.Name, op.Code)

	// Stack effect.
	in := op.StackIn(0)
	out := op.StackOut(0)
	if len(in) > 0 || len(out) > 0 {
		fmt.Fprintf(&sb, "Stack: `[%s]` → `[%s]`\n\n", strings.Join(in, ", "), strings.Join(out, ", "))
	}

	// Flags.
	var flags []string
	if op.Term {
		flags = append(flags, "terminal")
	}
	if op.Jump {
		kind := "conditional"
		if op.Unconditional {
			kind = "unconditional"
		}
		flags = append(flags, kind+" jump")
	}
	if op.JumpDest {
		flags = append(flags, "jump destination")
	}
	if op.HasImmediate {
		flags = append(flags, "has immediate")
	}
	if len(flags) > 0 {
		fmt.Fprintf(&sb, "Flags: %s\n\n", strings.Join(flags, ", "))
	}

	// Fork info.
	if forks := evm.ForksWhereOpAdded(op.Name); len(forks) > 0 {
		fmt.Fprintf(&sb, "Added in: %s", strings.Join(forks, ", "))
	}

	return sb.String()
}

func formatExprMacroHover(def *ast.ExpressionMacroDef) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "**Expression macro**\n\n```\n#define %s", def.Ident)
	if len(def.Params) > 0 {
		fmt.Fprintf(&sb, "(%s)", strings.Join(def.Params, ", "))
	}
	sb.WriteString(" = ...")
	sb.WriteString("\n```")
	return sb.String()
}

func formatInstrMacroHover(def *ast.InstructionMacroDef) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "**Instruction macro**\n\n```\n#define %%%s", def.Ident)
	if len(def.Params) > 0 {
		fmt.Fprintf(&sb, "(%s)", strings.Join(def.Params, ", "))
	}
	sb.WriteString(" { ... }")
	sb.WriteString("\n```")

	if def.StartComment != nil {
		sb.WriteString("\n\n")
		sb.WriteString(def.StartComment.InnerText())
	}
	return sb.String()
}

// builtinMacroDocs contains documentation for builtin expression macros.
var builtinMacroDocs = map[string]string{
	"len":       "**`.len(x)`** — Returns the byte length of value `x`.",
	"abs":       "**`.abs(x)`** — Returns the absolute value of `x`.",
	"intbits":   "**`.intbits(x)`** — Returns the number of bits needed to represent `x`.",
	"selector":  "**`.selector(\"signature\")`** — Returns the 4-byte function selector for the given ABI signature.",
	"keccak256": "**`.keccak256(x)`** — Returns the Keccak-256 hash of `x`.",
	"sha256":    "**`.sha256(x)`** — Returns the SHA-256 hash of `x`.",
	"address":   "**`.address(x)`** — Returns the checksummed 20-byte address from `x`.",
	"assemble":  "**`.assemble(\"file\")`** — Assembles the given file and returns its bytecode as a value.",
}

// directiveDocs contains documentation for directives.
var directiveDocs = map[string]string{
	"#define":   "**`#define`** — Defines an expression macro or instruction macro.\n\n- Expression: `#define NAME(params) = expr`\n- Instruction: `#define %NAME(params) { body }`",
	"#include":  "**`#include`** — Includes another `.eas` file.\n\n```\n#include \"filename.eas\"\n```\n\nDefinitions in the included file become available in the current file.",
	"#assemble": "**`#assemble`** — *(Deprecated)* Assembles a file and inlines its bytecode. Use `#bytes assemble(...)` instead.",
	"#pragma":   "**`#pragma`** — Sets a compiler option.\n\n```\n#pragma target \"forkname\"\n```\n\nSets the EVM instruction set for the program.",
	"#bytes":    "**`#bytes`** — Inserts raw bytes into the output.\n\n```\n#bytes 0x010203\n#bytes \"string\"\n#bytes name: expression\n```",
}
