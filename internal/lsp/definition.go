package lsp

import (
	"github.com/fjl/geas/internal/ast"
)

// computeDefinition finds the definition of the symbol at the given position.
func computeDefinition(doc *document, pos Position) *Location {
	tok, ok := tokenAtPosition(doc.Content, pos)
	if !ok || doc.AST == nil {
		return nil
	}

	switch tok.Type {
	case ast.TokenLabelRef, ast.TokenDottedLabelRef:
		lref := &ast.LabelRefExpr{
			Ident:  tok.Text,
			Dotted: tok.Type == ast.TokenDottedLabelRef,
		}
		ldef, _ := doc.AST.LookupLabel(lref)
		if ldef == nil {
			return nil
		}
		defPos := ldef.Position()
		return &Location{
			URI:   pathToURI(defPos.File),
			Range: posToRange(defPos),
		}

	case ast.TokenInstMacroIdent:
		def, _ := doc.AST.LookupInstrMacro(tok.Text)
		if def == nil {
			return nil
		}
		defPos := def.Position()
		return &Location{
			URI:   pathToURI(defPos.File),
			Range: posToRange(defPos),
		}

	case ast.TokenIdentifier:
		// Could be an expression macro reference.
		def, _ := doc.AST.LookupExprMacro(tok.Text)
		if def == nil {
			return nil
		}
		defPos := def.Position()
		return &Location{
			URI:   pathToURI(defPos.File),
			Range: posToRange(defPos),
		}
	}

	return nil
}
