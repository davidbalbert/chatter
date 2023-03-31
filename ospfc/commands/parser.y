%{

package commands

type astType int

const (
	_ astType = iota
	astCommand
	astTokens
	astToken
)

type ast struct {
	_type    astType
	value    string
	children []*ast
}



%}

%union {
	node *ast
	s string
}

%type <node> command
%type <node> tokens
%type <node> token

%token <s> LITERAL VARIABLE
%token <s> WS // ignored

%%

command:
	tokens { (yylex).(*lexer).result = newNode(astCommand, "", $1) }

tokens:
	tokens token { $1.children = append($1.children, $2) }
	| token { $$ = newNode(astTokens, "", $1) }

token: LITERAL { $$ = newNode(astToken, $1) }

%%

func newNode(_type astType, value string, children ...*ast) *ast {
	return &ast{_type: _type, value: value, children: children}
}
