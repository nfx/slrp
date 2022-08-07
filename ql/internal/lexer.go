package internal

import (
	"fmt"
	"text/scanner"
	"time"
)

type Lexer struct {
	scanner.Scanner
	prevTok int
	err     error
}

func (l *Lexer) Lex(lval *yySymType) int {
	token := l.Scan()
	lit := l.TokenText()
	lval.literal = lit
	tok := l.Match(token, lval)
	l.prevTok = tok
	return tok
}

func (l *Lexer) Match(token rune, lval *yySymType) int {
	tok := int(token)
	switch tok {
	case scanner.Int, scanner.Float:
		return NUMBER
	case scanner.String:
		return STRING
	case scanner.Ident:
		return l.ident(lval)
	default:
		if token == '<' && l.Peek() == '>' {
			l.Next()
			lval.literal = "<>"
			return NEQ
		}
		if token == '!' && l.Peek() == '=' {
			l.Next()
			lval.literal = "!="
			return NEQ
		}
		switch lval.literal {
		case "!":
			return NOT
		case ":", "=":
			return EQ
		}
	}
	return tok
}

func (l *Lexer) ident(lval *yySymType) int {
	switch lit := lval.literal; {
	case lit == "NOT":
		return NOT
	case lit == "AND":
		return AND
	case lit == "ORDER":
		return ORDER
	case lit == "BY":
		return BY
	case lit == "OR":
		return OR
	case lit == "ASC":
		return ASC
	case lit == "DESC":
		return DESC
	case lit == "LIMIT":
		return LIMIT
	case lit == "w" && l.prevTok == NUMBER:
		lval.dur = 7 * 24 * time.Hour
		return DUR
	case lit == "d" && l.prevTok == NUMBER:
		lval.dur = 24 * time.Hour
		return DUR
	case lit == "h" && l.prevTok == NUMBER:
		lval.dur = time.Hour
		return DUR
	case lit == "m" && l.prevTok == NUMBER:
		lval.dur = time.Minute
		return DUR
	case lit == "s" && l.prevTok == NUMBER:
		lval.dur = time.Second
		return DUR
	default:
		return IDENT
	}
}

func (l *Lexer) Error(err string) {
	l.err = fmt.Errorf(err)
}
