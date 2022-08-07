%{
package internal

import (
  "github.com/nfx/slrp/ql/ast"
  "strconv"
  "strings"
  "time"
)

%}

%union{
  query   ast.Query
  literal string
  expr    ast.Node
  dur     time.Duration
  sort    ast.Sort
  orderBy ast.OrderBy
  dir     bool
  num     int
}

%type<query>    query
%type<expr>     expr '(' 
%type<sort>     sort order
%type<orderBy>  orderBy
%type<dir>      direction
%type<num>      limit

%token<literal>	NUMBER IDENT STRING NOT AND OR EQ NEQ ORDER BY LIMIT
%token<dur>     DUR
%token<dir>     ASC DESC

%left   OR
%left   AND
%right  NOT
%left   '<' '>' '~'
%left   EQ NEQ
%left   '('

%start query

%%

query: 
  expr sort limit {
    $$ = ast.Query{$1, $2, $3}
  }

limit: 
  {}
  | LIMIT NUMBER {
    v, _ := strconv.ParseInt($2, 10, 32)
    $$ = int(v)
  }

sort:
  {}
  | ORDER BY order {
    $$ = $3
  }

order: 
  orderBy {
    // create single-item ORDER BY
    $$ = append($$, $1)
  }
  | order ',' orderBy {
    // add to existing ORDER BY
    $$ = append($1, $3)
  }

orderBy:
  IDENT direction {
    $$ = ast.OrderBy{$1, $2}
  }

direction: { $$ = true }
  | ASC { $$ = true }
  | DESC { $$ = false }

expr: 
  NOT expr {
    $$ = ast.Not{$2}
  }
  | expr '<' expr {
    $$ = ast.LessThan{$1, $3}
  }
  | expr '>' expr {
    $$ = ast.GreaterThan{$1, $3}
  }
  | expr '~' expr {
    $$ = ast.Contains{$1, $3}
  }
  | expr AND expr {
    $$ = ast.And{$1, $3}
  }
  | expr OR expr {
    $$ = ast.Or{$1, $3}
  }
  | expr EQ expr {
    $$ = ast.Equals{$1, $3}
  }
  | expr NEQ expr {
    $$ = ast.Not{ast.Equals{$1, $3}}
  }
  | '(' expr ')' {
    $$ = $2
  }
  | NUMBER DUR {
    v, _ := strconv.ParseFloat($1, 64)
    $$ = ast.Duration(time.Duration(v) * $2)
  }
  | NUMBER {
    v, _ := strconv.ParseFloat($1, 64)
    $$ = ast.Number(v)
  }
  | IDENT {
    $$ = ast.Ident($1)
  }
  | STRING {
    $$ = ast.String(strings.Trim($1, "`'\""))
  }

%%