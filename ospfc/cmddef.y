%{

package main

%}

%union {
	tokens []string
	s string
}

%type <tokens> command
%type <tokens> tokens
%type <s> token

%token <s> LITERAL
%token <s> VARIABLE

%%

command:
	tokens { cmddeflex.(*cmddefLex).result = $$ }

tokens:
	tokens token { $$ = append($1, $2) }
	| token { $$ = []string{$1} }

token: LITERAL

%%
