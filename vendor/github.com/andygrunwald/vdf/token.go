package vdf

// Token is our own type that represents a single token
// to work with during parsing
type Token int

const (
	_ Token = iota

	// Illegal represents a token that we
	// don`t know in contect of the VDF format
	Illegal

	// EOF represents the End of File token.
	// This is used if the file is end
	EOF

	// WS represents a whitespace.
	// This can be a space or a tab.
	WS

	// EOL represents a line ending.
	// This can be a \n or a \r.
	EOL

	// Ident represents a key or a value.
	// Typically this is a simple string
	Ident

	// CurlyBraceOpen represents a open curly brace "{"
	CurlyBraceOpen

	// CurlyBraceClose represents a close curly brace "}"
	CurlyBraceClose

	// QuotationMark represents a quote mark '"'
	QuotationMark

	// EscapeSequence represents an escape character "\"
	EscapeSequence

	// CommentDoubleSlash represents a comment prefix with a double slash "//"
	CommentDoubleSlash
)

var eof = rune(0)
