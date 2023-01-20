package vdf

import (
	"bufio"
	"bytes"
	"io"
)

// Scanner represents a lexical scanner.
type Scanner struct {
	r *bufio.Reader
}

// NewScanner returns a new instance of Scanner.
func NewScanner(r io.Reader) *Scanner {
	return &Scanner{r: bufio.NewReader(r)}
}

// read reads the next rune from the buffered reader.
// Returns the rune(0) if an error occurs (or io.EOF is returned).
func (s *Scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	return ch
}

// unread places the previously read rune back on the reader.
func (s *Scanner) unread() {
	_ = s.r.UnreadRune()
}

// Scan returns the next token and literal value.
func (s *Scanner) Scan(respectWhitespace bool) (tok Token, lit string) {
	// Read the next rune.
	ch := s.read()

	// If we see whitespace then consume all contiguous whitespace.
	if !respectWhitespace && (isWhitespace(ch) || isLineEnding(ch)) {
		s.unread()
		return s.scanWhitespace()

		// If we see a whitespace, return it
	} else if respectWhitespace && isWhitespace(ch) {
		return WS, string(ch)

		// If we see a line ending, return it
	} else if respectWhitespace && isLineEnding(ch) {
		return EOL, string(ch)

		// If we see a letter then consume as an ident or reserved word.
	} else if isLetter(ch) || isDigit(ch) {
		s.unread()
		return s.scanIdent()

		// If we see a "//" this line is a comment
	} else if isComment(ch, s) {
		return CommentDoubleSlash, string(ch) + string(ch)
	}

	// Otherwise read the individual character.
	switch ch {
	case eof:
		return EOF, ""
	case '\\':
		return EscapeSequence, string(ch)
	case '{':
		return CurlyBraceOpen, string(ch)
	case '}':
		return CurlyBraceClose, string(ch)
	case '"':
		return QuotationMark, string(ch)
	}

	return Illegal, string(ch)
}

// scanWhitespace consumes the current rune and all contiguous whitespace.
func (s *Scanner) scanWhitespace() (Token, string) {
	// Create a buffer and read the current character into it.
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	// Read every subsequent whitespace character into the buffer.
	// Non-whitespace characters and EOF will cause the loop to exit.
	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isWhitespace(ch) {
			s.unread()
			break
		} else {
			buf.WriteRune(ch)
		}
	}

	return WS, buf.String()
}

// scanIdent consumes the current rune and all contiguous ident runes.
func (s *Scanner) scanIdent() (tok Token, lit string) {
	// Create a buffer and read the current character into it.
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	// Read every subsequent ident character into the buffer.
	// Non-ident characters and EOF will cause the loop to exit.
	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isLetter(ch) && !isDigit(ch) && ch != '_' {
			s.unread()
			break
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}

	// Otherwise return as a regular identifier.
	return Ident, buf.String()
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t'
}

func isLineEnding(ch rune) bool {
	return ch == '\n' || ch == '\r'
}

func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

// isDigit returns true if the rune is a digit.
func isDigit(ch rune) bool {
	return (ch >= '0' && ch <= '9')
}

// isComment returns true if this line starts with a comment ("//")
func isComment(ch rune, s *Scanner) bool {
	if ch != '/' {
		return false
	}

	nextRune := s.read()
	if nextRune != '/' {
		s.unread()
		return false
	}

	return true
}
