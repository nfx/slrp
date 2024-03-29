// Code generated by goyacc -o parser.go parser.y. DO NOT EDIT.

//line parser.y:2
package internal

import __yyfmt__ "fmt"

//line parser.y:2

import (
	"github.com/nfx/slrp/ql/ast"
	"strconv"
	"strings"
	"time"
)

//line parser.y:13
type yySymType struct {
	yys     int
	query   ast.Query
	literal string
	expr    ast.Node
	dur     time.Duration
	sort    ast.Sort
	orderBy ast.OrderBy
	dir     bool
	num     int
}

const NUMBER = 57346
const IDENT = 57347
const STRING = 57348
const NOT = 57349
const AND = 57350
const OR = 57351
const EQ = 57352
const NEQ = 57353
const ORDER = 57354
const BY = 57355
const LIMIT = 57356
const DUR = 57357
const ASC = 57358
const DESC = 57359

var yyToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"'('",
	"NUMBER",
	"IDENT",
	"STRING",
	"NOT",
	"AND",
	"OR",
	"EQ",
	"NEQ",
	"ORDER",
	"BY",
	"LIMIT",
	"DUR",
	"ASC",
	"DESC",
	"'<'",
	"'>'",
	"'~'",
	"','",
	"')'",
}

var yyStatenames = [...]string{}

const yyEofCode = 1
const yyErrCode = 2
const yyInitialStackSize = 16

//line parser.y:126

//line yacctab:1
var yyExca = [...]int8{
	-1, 1,
	1, -1,
	-2, 0,
}

const yyPrivate = 57344

const yyLast = 59

var yyAct = [...]int8{
	33, 12, 13, 14, 15, 35, 12, 13, 14, 15,
	16, 9, 10, 11, 19, 30, 9, 10, 11, 12,
	21, 14, 15, 34, 14, 15, 37, 38, 29, 9,
	10, 11, 9, 10, 11, 2, 39, 14, 15, 17,
	18, 31, 20, 36, 32, 22, 23, 24, 25, 26,
	27, 28, 4, 5, 6, 7, 3, 8, 1,
}

var yyPact = [...]int16{
	48, -1000, -3, 48, 48, -2, -1000, -1000, 5, 48,
	48, 48, 48, 48, 48, 48, 14, 13, -8, -1000,
	-1000, 36, 26, 26, 26, 13, 10, -1000, -1000, 17,
	-1000, -1000, -17, -1000, 9, 17, -1000, -1000, -1000, -1000,
}

var yyPgo = [...]int8{
	0, 58, 35, 57, 44, 0, 43, 42,
}

var yyR1 = [...]int8{
	0, 1, 7, 7, 3, 3, 4, 4, 5, 6,
	6, 6, 2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2,
}

var yyR2 = [...]int8{
	0, 3, 0, 2, 0, 3, 1, 3, 2, 0,
	1, 1, 2, 3, 3, 3, 3, 3, 3, 3,
	3, 2, 1, 1, 1,
}

var yyChk = [...]int16{
	-1000, -1, -2, 8, 4, 5, 6, 7, -3, 19,
	20, 21, 9, 10, 11, 12, 13, -2, -2, 16,
	-7, 15, -2, -2, -2, -2, -2, -2, -2, 14,
	23, 5, -4, -5, 6, 22, -6, 17, 18, -5,
}

var yyDef = [...]int8{
	0, -2, 4, 0, 0, 22, 23, 24, 2, 0,
	0, 0, 0, 0, 0, 0, 0, 12, 0, 21,
	1, 0, 13, 14, 15, 16, 17, 18, 19, 0,
	20, 3, 5, 6, 9, 0, 8, 10, 11, 7,
}

var yyTok1 = [...]int8{
	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	4, 23, 3, 3, 22, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	19, 3, 20, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 21,
}

var yyTok2 = [...]int8{
	2, 3, 5, 6, 7, 8, 9, 10, 11, 12,
	13, 14, 15, 16, 17, 18,
}

var yyTok3 = [...]int8{
	0,
}

var yyErrorMessages = [...]struct {
	state int
	token int
	msg   string
}{}

//line yaccpar:1

/*	parser for yacc output	*/

var (
	yyDebug        = 0
	yyErrorVerbose = false
)

type yyLexer interface {
	Lex(lval *yySymType) int
	Error(s string)
}

type yyParser interface {
	Parse(yyLexer) int
	Lookahead() int
}

type yyParserImpl struct {
	lval  yySymType
	stack [yyInitialStackSize]yySymType
	char  int
}

func (p *yyParserImpl) Lookahead() int {
	return p.char
}

func yyNewParser() yyParser {
	return &yyParserImpl{}
}

const yyFlag = -1000

func yyTokname(c int) string {
	if c >= 1 && c-1 < len(yyToknames) {
		if yyToknames[c-1] != "" {
			return yyToknames[c-1]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func yyStatname(s int) string {
	if s >= 0 && s < len(yyStatenames) {
		if yyStatenames[s] != "" {
			return yyStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func yyErrorMessage(state, lookAhead int) string {
	const TOKSTART = 4

	if !yyErrorVerbose {
		return "syntax error"
	}

	for _, e := range yyErrorMessages {
		if e.state == state && e.token == lookAhead {
			return "syntax error: " + e.msg
		}
	}

	res := "syntax error: unexpected " + yyTokname(lookAhead)

	// To match Bison, suggest at most four expected tokens.
	expected := make([]int, 0, 4)

	// Look for shiftable tokens.
	base := int(yyPact[state])
	for tok := TOKSTART; tok-1 < len(yyToknames); tok++ {
		if n := base + tok; n >= 0 && n < yyLast && int(yyChk[int(yyAct[n])]) == tok {
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}
	}

	if yyDef[state] == -2 {
		i := 0
		for yyExca[i] != -1 || int(yyExca[i+1]) != state {
			i += 2
		}

		// Look for tokens that we accept or reduce.
		for i += 2; yyExca[i] >= 0; i += 2 {
			tok := int(yyExca[i])
			if tok < TOKSTART || yyExca[i+1] == 0 {
				continue
			}
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}

		// If the default action is to accept or reduce, give up.
		if yyExca[i+1] != 0 {
			return res
		}
	}

	for i, tok := range expected {
		if i == 0 {
			res += ", expecting "
		} else {
			res += " or "
		}
		res += yyTokname(tok)
	}
	return res
}

func yylex1(lex yyLexer, lval *yySymType) (char, token int) {
	token = 0
	char = lex.Lex(lval)
	if char <= 0 {
		token = int(yyTok1[0])
		goto out
	}
	if char < len(yyTok1) {
		token = int(yyTok1[char])
		goto out
	}
	if char >= yyPrivate {
		if char < yyPrivate+len(yyTok2) {
			token = int(yyTok2[char-yyPrivate])
			goto out
		}
	}
	for i := 0; i < len(yyTok3); i += 2 {
		token = int(yyTok3[i+0])
		if token == char {
			token = int(yyTok3[i+1])
			goto out
		}
	}

out:
	if token == 0 {
		token = int(yyTok2[1]) /* unknown char */
	}
	if yyDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", yyTokname(token), uint(char))
	}
	return char, token
}

func yyParse(yylex yyLexer) int {
	return yyNewParser().Parse(yylex)
}

func (yyrcvr *yyParserImpl) Parse(yylex yyLexer) int {
	var yyn int
	var yyVAL yySymType
	var yyDollar []yySymType
	_ = yyDollar // silence set and not used
	yyS := yyrcvr.stack[:]

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yystate := 0
	yyrcvr.char = -1
	yytoken := -1 // yyrcvr.char translated into internal numbering
	defer func() {
		// Make sure we report no lookahead when not parsing.
		yystate = -1
		yyrcvr.char = -1
		yytoken = -1
	}()
	yyp := -1
	goto yystack

ret0:
	return 0

ret1:
	return 1

yystack:
	/* put a state and value onto the stack */
	if yyDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", yyTokname(yytoken), yyStatname(yystate))
	}

	yyp++
	if yyp >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyS[yyp] = yyVAL
	yyS[yyp].yys = yystate

yynewstate:
	yyn = int(yyPact[yystate])
	if yyn <= yyFlag {
		goto yydefault /* simple state */
	}
	if yyrcvr.char < 0 {
		yyrcvr.char, yytoken = yylex1(yylex, &yyrcvr.lval)
	}
	yyn += yytoken
	if yyn < 0 || yyn >= yyLast {
		goto yydefault
	}
	yyn = int(yyAct[yyn])
	if int(yyChk[yyn]) == yytoken { /* valid shift */
		yyrcvr.char = -1
		yytoken = -1
		yyVAL = yyrcvr.lval
		yystate = yyn
		if Errflag > 0 {
			Errflag--
		}
		goto yystack
	}

yydefault:
	/* default state action */
	yyn = int(yyDef[yystate])
	if yyn == -2 {
		if yyrcvr.char < 0 {
			yyrcvr.char, yytoken = yylex1(yylex, &yyrcvr.lval)
		}

		/* look through exception table */
		xi := 0
		for {
			if yyExca[xi+0] == -1 && int(yyExca[xi+1]) == yystate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			yyn = int(yyExca[xi+0])
			if yyn < 0 || yyn == yytoken {
				break
			}
		}
		yyn = int(yyExca[xi+1])
		if yyn < 0 {
			goto ret0
		}
	}
	if yyn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			yylex.Error(yyErrorMessage(yystate, yytoken))
			Nerrs++
			if yyDebug >= 1 {
				__yyfmt__.Printf("%s", yyStatname(yystate))
				__yyfmt__.Printf(" saw %s\n", yyTokname(yytoken))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for yyp >= 0 {
				yyn = int(yyPact[yyS[yyp].yys]) + yyErrCode
				if yyn >= 0 && yyn < yyLast {
					yystate = int(yyAct[yyn]) /* simulate a shift of "error" */
					if int(yyChk[yystate]) == yyErrCode {
						goto yystack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if yyDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", yyS[yyp].yys)
				}
				yyp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if yyDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", yyTokname(yytoken))
			}
			if yytoken == yyEofCode {
				goto ret1
			}
			yyrcvr.char = -1
			yytoken = -1
			goto yynewstate /* try again in the same state */
		}
	}

	/* reduction by production yyn */
	if yyDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", yyn, yyStatname(yystate))
	}

	yynt := yyn
	yypt := yyp
	_ = yypt // guard against "declared and not used"

	yyp -= int(yyR2[yyn])
	// yyp is now the index of $0. Perform the default action. Iff the
	// reduced production is ε, $1 is possibly out of range.
	if yyp+1 >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyVAL = yyS[yyp+1]

	/* consult goto table to find next state */
	yyn = int(yyR1[yyn])
	yyg := int(yyPgo[yyn])
	yyj := yyg + yyS[yyp].yys + 1

	if yyj >= yyLast {
		yystate = int(yyAct[yyg])
	} else {
		yystate = int(yyAct[yyj])
		if int(yyChk[yystate]) != -yyn {
			yystate = int(yyAct[yyg])
		}
	}
	// dummy call; replaced with literal code
	switch yynt {

	case 1:
		yyDollar = yyS[yypt-3 : yypt+1]
//line parser.y:47
		{
			yyVAL.query = ast.Query{yyDollar[1].expr, yyDollar[2].sort, yyDollar[3].num}
		}
	case 2:
		yyDollar = yyS[yypt-0 : yypt+1]
//line parser.y:52
		{
		}
	case 3:
		yyDollar = yyS[yypt-2 : yypt+1]
//line parser.y:53
		{
			v, _ := strconv.ParseInt(yyDollar[2].literal, 10, 32)
			yyVAL.num = int(v)
		}
	case 4:
		yyDollar = yyS[yypt-0 : yypt+1]
//line parser.y:59
		{
		}
	case 5:
		yyDollar = yyS[yypt-3 : yypt+1]
//line parser.y:60
		{
			yyVAL.sort = yyDollar[3].sort
		}
	case 6:
		yyDollar = yyS[yypt-1 : yypt+1]
//line parser.y:65
		{
			// create single-item ORDER BY
			yyVAL.sort = append(yyVAL.sort, yyDollar[1].orderBy)
		}
	case 7:
		yyDollar = yyS[yypt-3 : yypt+1]
//line parser.y:69
		{
			// add to existing ORDER BY
			yyVAL.sort = append(yyDollar[1].sort, yyDollar[3].orderBy)
		}
	case 8:
		yyDollar = yyS[yypt-2 : yypt+1]
//line parser.y:75
		{
			yyVAL.orderBy = ast.OrderBy{yyDollar[1].literal, yyDollar[2].dir}
		}
	case 9:
		yyDollar = yyS[yypt-0 : yypt+1]
//line parser.y:79
		{
			yyVAL.dir = true
		}
	case 10:
		yyDollar = yyS[yypt-1 : yypt+1]
//line parser.y:80
		{
			yyVAL.dir = true
		}
	case 11:
		yyDollar = yyS[yypt-1 : yypt+1]
//line parser.y:81
		{
			yyVAL.dir = false
		}
	case 12:
		yyDollar = yyS[yypt-2 : yypt+1]
//line parser.y:84
		{
			yyVAL.expr = ast.Not{yyDollar[2].expr}
		}
	case 13:
		yyDollar = yyS[yypt-3 : yypt+1]
//line parser.y:87
		{
			yyVAL.expr = ast.LessThan{yyDollar[1].expr, yyDollar[3].expr}
		}
	case 14:
		yyDollar = yyS[yypt-3 : yypt+1]
//line parser.y:90
		{
			yyVAL.expr = ast.GreaterThan{yyDollar[1].expr, yyDollar[3].expr}
		}
	case 15:
		yyDollar = yyS[yypt-3 : yypt+1]
//line parser.y:93
		{
			yyVAL.expr = ast.Contains{yyDollar[1].expr, yyDollar[3].expr}
		}
	case 16:
		yyDollar = yyS[yypt-3 : yypt+1]
//line parser.y:96
		{
			yyVAL.expr = ast.And{yyDollar[1].expr, yyDollar[3].expr}
		}
	case 17:
		yyDollar = yyS[yypt-3 : yypt+1]
//line parser.y:99
		{
			yyVAL.expr = ast.Or{yyDollar[1].expr, yyDollar[3].expr}
		}
	case 18:
		yyDollar = yyS[yypt-3 : yypt+1]
//line parser.y:102
		{
			yyVAL.expr = ast.Equals{yyDollar[1].expr, yyDollar[3].expr}
		}
	case 19:
		yyDollar = yyS[yypt-3 : yypt+1]
//line parser.y:105
		{
			yyVAL.expr = ast.Not{ast.Equals{yyDollar[1].expr, yyDollar[3].expr}}
		}
	case 20:
		yyDollar = yyS[yypt-3 : yypt+1]
//line parser.y:108
		{
			yyVAL.expr = yyDollar[2].expr
		}
	case 21:
		yyDollar = yyS[yypt-2 : yypt+1]
//line parser.y:111
		{
			v, _ := strconv.ParseFloat(yyDollar[1].literal, 64)
			yyVAL.expr = ast.Duration(time.Duration(v) * yyDollar[2].dur)
		}
	case 22:
		yyDollar = yyS[yypt-1 : yypt+1]
//line parser.y:115
		{
			v, _ := strconv.ParseFloat(yyDollar[1].literal, 64)
			yyVAL.expr = ast.Number(v)
		}
	case 23:
		yyDollar = yyS[yypt-1 : yypt+1]
//line parser.y:119
		{
			yyVAL.expr = ast.Ident(yyDollar[1].literal)
		}
	case 24:
		yyDollar = yyS[yypt-1 : yypt+1]
//line parser.y:122
		{
			yyVAL.expr = ast.String(strings.Trim(yyDollar[1].literal, "`'\""))
		}
	}
	goto yystack /* stack new state and value */
}
