package ast

// Token is an exported token type for use by the LSP server.
type Token struct {
	Text string
	Line int
	Col  int
	Type TokenTypeExported
}

// TokenTypeExported is the exported version of tokenType.
type TokenTypeExported byte

// Exported token type constants.
const (
	TokenEOF                TokenTypeExported = TokenTypeExported(eof)
	TokenLineStart          TokenTypeExported = TokenTypeExported(lineStart)
	TokenLineEnd            TokenTypeExported = TokenTypeExported(lineEnd)
	TokenInvalid            TokenTypeExported = TokenTypeExported(invalidToken)
	TokenIdentifier         TokenTypeExported = TokenTypeExported(identifier)
	TokenDottedIdentifier   TokenTypeExported = TokenTypeExported(dottedIdentifier)
	TokenVariableIdentifier TokenTypeExported = TokenTypeExported(variableIdentifier)
	TokenLabelRef           TokenTypeExported = TokenTypeExported(labelRef)
	TokenDottedLabelRef     TokenTypeExported = TokenTypeExported(dottedLabelRef)
	TokenLabel              TokenTypeExported = TokenTypeExported(label)
	TokenDottedLabel        TokenTypeExported = TokenTypeExported(dottedLabel)
	TokenNumberLiteral      TokenTypeExported = TokenTypeExported(numberLiteral)
	TokenStringLiteral      TokenTypeExported = TokenTypeExported(stringLiteral)
	TokenOpenParen          TokenTypeExported = TokenTypeExported(openParen)
	TokenCloseParen         TokenTypeExported = TokenTypeExported(closeParen)
	TokenComma              TokenTypeExported = TokenTypeExported(comma)
	TokenDirective          TokenTypeExported = TokenTypeExported(directive)
	TokenInstMacroIdent     TokenTypeExported = TokenTypeExported(instMacroIdent)
	TokenOpenBrace          TokenTypeExported = TokenTypeExported(openBrace)
	TokenCloseBrace         TokenTypeExported = TokenTypeExported(closeBrace)
	TokenOpenBracket        TokenTypeExported = TokenTypeExported(openBracket)
	TokenCloseBracket       TokenTypeExported = TokenTypeExported(closeBracket)
	TokenEquals             TokenTypeExported = TokenTypeExported(equals)
	TokenArith              TokenTypeExported = TokenTypeExported(arith)
	TokenComment            TokenTypeExported = TokenTypeExported(comment)
)

// LexAll tokenizes the source and returns all tokens.
func LexAll(source []byte) []Token {
	ch := runLexer(source)
	var tokens []Token
	for tok := range ch {
		tokens = append(tokens, Token{
			Text: tok.text,
			Line: tok.line,
			Col:  tok.col,
			Type: TokenTypeExported(tok.typ),
		})
	}
	return tokens
}
