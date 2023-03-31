%{

package main

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
	tokens { (cmddeflex).(*lexer).result = newNode(astCommand, "", $1) }

tokens:
	tokens token { $1.children = append($1.children, $2) }
	| token { $$ = newNode(astTokens, "", $1) }

token: LITERAL { $$ = newNode(astToken, $1) }

%%
