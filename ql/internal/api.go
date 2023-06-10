package internal

import (
	"fmt"
	"strings"
	"text/scanner"

	"github.com/nfx/slrp/ql/ast"
)

//go:generate goyacc -o parser.go parser.y

func Parse(query string) (*ast.Query, error) {
	if query == "" {
		return &ast.Query{
			Limit:  20,
			Filter: ast.True,
		}, nil
	}
	lexer := &Lexer{
		Scanner: scanner.Scanner{
			IsIdentRune: func(ch rune, i int) bool {
				lower := ('a' - 'A') | ch
				return 'a' <= lower && lower <= 'z'
			},
		},
	}
	lexer.Init(strings.NewReader(query))
	parser := &yyParserImpl{}
	yyErrorVerbose = true
	parser.Parse(lexer)
	if lexer.err != nil {
		return nil, parseError(lexer, query, parser.lval.literal)
	}
	dollarOne := parser.stack[1].query
	return &dollarOne, nil
}

const explainBuffer = 5

func parseError(lexer *Lexer, query string, literal string) *ParseError {
	expStart := max(0, lexer.Offset-explainBuffer)
	left := query[expStart:lexer.Offset]
	litEnd := lexer.Offset + len(literal)
	expEnd := litEnd + min(explainBuffer, len(query)-litEnd)
	right := query[litEnd:expEnd]
	expl := strings.Builder{}
	if expStart > 0 {
		expl.WriteString("..")
	}
	expl.WriteString(left)
	expl.WriteString("<<<")
	expl.WriteString(literal)
	expl.WriteString(">>>")
	expl.WriteString(right)
	if expEnd < len(query) {
		expl.WriteString("..")
	}
	return &ParseError{
		Message:     lexer.err.Error(),
		Explanation: strings.Trim(expl.String(), " "),
		Offset:      lexer.Offset,
		Length:      len(literal),
		Line:        lexer.Line,
		Column:      lexer.Column,
	}
}

type ParseError struct {
	Message     string
	Explanation string
	Offset      int
	Length      int
	Line        int
	Column      int
}

func (err *ParseError) Error() string {
	return fmt.Sprintf("%s: %s", err.Message, err.Explanation)
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
