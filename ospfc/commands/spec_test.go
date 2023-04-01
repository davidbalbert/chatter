package commands

import (
	"strings"
	"testing"
)

func TestParseFork(t *testing.T) {
	spec, err := parseSpec("fork")
	if err != nil {
		t.Fatal(err)
	}

	if spec.typeName != "fork" {
		t.Fatalf("expected fork, got %q", spec.typeName)
	}

	if spec.arg != "" {
		t.Fatalf("expected empty arg, got %q", spec.arg)
	}

	if len(spec.children) != 0 {
		t.Fatalf("expected no children, got %d", len(spec.children))
	}

	if spec.id != 0 {
		t.Fatalf("expected id 0, got %d", spec.id)
	}
}

func TestParseJoin(t *testing.T) {
	spec, err := parseSpec("join")
	if err != nil {
		t.Fatal(err)
	}

	if spec.typeName != "join" {
		t.Fatalf("expected join, got %q", spec.typeName)
	}

	if spec.arg != "" {
		t.Fatalf("expected empty arg, got %q", spec.arg)
	}

	if len(spec.children) != 0 {
		t.Fatalf("expected no children, got %d", len(spec.children))
	}

	if spec.id != 0 {
		t.Fatalf("expected id 0, got %d", spec.id)
	}
}

func TestParseJoinWithID(t *testing.T) {
	spec, err := parseSpec("join.1")
	if err != nil {
		t.Fatal(err)
	}

	if spec.typeName != "join" {
		t.Fatalf("expected join, got %q", spec.typeName)
	}

	if spec.arg != "" {
		t.Fatalf("expected empty arg, got %q", spec.arg)
	}

	if len(spec.children) != 0 {
		t.Fatalf("expected no children, got %d", len(spec.children))
	}

	if spec.id != 1 {
		t.Fatalf("expected id 1, got %d", spec.id)
	}
}

func TestParseLiteral(t *testing.T) {
	spec, err := parseSpec("literal:foo")
	if err != nil {
		t.Fatal(err)
	}

	if spec.typeName != "literal" {
		t.Fatalf("expected literal, got %q", spec.typeName)
	}

	if spec.arg != "foo" {
		t.Fatalf("expected arg foo, got %q", spec.arg)
	}

	if len(spec.children) != 0 {
		t.Fatalf("expected no children, got %d", len(spec.children))
	}

	if spec.id != 0 {
		t.Fatalf("expected id 0, got %d", spec.id)
	}
}

func TestParseLiteralMissingValue(t *testing.T) {
	_, err := parseSpec("literal")
	if err == nil {
		t.Fatal("expected error")
	}

	_, err = parseSpec("literal:")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseArgumentString(t *testing.T) {
	spec, err := parseSpec("argument:string")
	if err != nil {
		t.Fatal(err)
	}

	if spec.typeName != "argument" {
		t.Fatalf("expected argument, got %q", spec.typeName)
	}

	if spec.arg != "string" {
		t.Fatalf("expected arg foo, got %q", spec.arg)
	}

	if len(spec.children) != 0 {
		t.Fatalf("expected no children, got %d", len(spec.children))
	}

	if spec.id != 0 {
		t.Fatalf("expected id 0, got %d", spec.id)
	}
}

func TestParseArgumentIPv4(t *testing.T) {
	spec, err := parseSpec("argument:ipv4")
	if err != nil {
		t.Fatal(err)
	}

	if spec.typeName != "argument" {
		t.Fatalf("expected argument, got %q", spec.typeName)
	}

	if spec.arg != "ipv4" {
		t.Fatalf("expected arg foo, got %q", spec.arg)
	}

	if len(spec.children) != 0 {
		t.Fatalf("expected no children, got %d", len(spec.children))
	}

	if spec.id != 0 {
		t.Fatalf("expected id 0, got %d", spec.id)
	}
}

func TestParseArgumentIPv6(t *testing.T) {
	spec, err := parseSpec("argument:ipv6")
	if err != nil {
		t.Fatal(err)
	}

	if spec.typeName != "argument" {
		t.Fatalf("expected argument, got %q", spec.typeName)
	}

	if spec.arg != "ipv6" {
		t.Fatalf("expected arg foo, got %q", spec.arg)
	}

	if len(spec.children) != 0 {
		t.Fatalf("expected no children, got %d", len(spec.children))
	}

	if spec.id != 0 {
		t.Fatalf("expected id 0, got %d", spec.id)
	}
}

func TestParseArgumentBadType(t *testing.T) {
	_, err := parseSpec("argument:foo")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseArgumentMissingType(t *testing.T) {
	_, err := parseSpec("argument")
	if err == nil {
		t.Fatal("expected error")
	}

	_, err = parseSpec("argument:")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLiteralWithChild(t *testing.T) {
	s, err := parseSpec("literal:foo[literal:bar]")
	if err != nil {
		t.Fatal(err)
	}

	if s == nil {
		t.Fatal("expected spec")
	}

	if s.typeName != "literal" {
		t.Fatalf("expected literal, got %q", s.typeName)
	}

	if s.arg != "foo" {
		t.Fatalf("expected arg foo, got %q", s.arg)
	}

	if s.id != 0 {
		t.Fatalf("expected id 0, got %d", s.id)
	}

	if len(s.children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(s.children))
	}

	if s.children[0].typeName != "literal" {
		t.Fatalf("expected literal, got %q", s.children[0].typeName)
	}

	if s.children[0].arg != "bar" {
		t.Fatalf("expected arg bar, got %q", s.children[0].arg)
	}

	if s.children[0].id != 0 {
		t.Fatalf("expected id 0, got %d", s.children[0].id)
	}

	if len(s.children[0].children) != 0 {
		t.Fatalf("expected no children, got %d", len(s.children[0].children))
	}
}

func TestLiteralWithChildAndWhitespace(t *testing.T) {
	s := `
	literal:foo[
		literal:bar
	]
	`

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	if spec.typeName != "literal" {
		t.Fatalf("expected literal, got %q", spec.typeName)
	}

	if spec.arg != "foo" {
		t.Fatalf("expected arg foo, got %q", spec.arg)
	}

	if spec.id != 0 {
		t.Fatalf("expected id 0, got %d", spec.id)
	}

	if len(spec.children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(spec.children))
	}

	if spec.children[0].typeName != "literal" {
		t.Fatalf("expected literal, got %q", spec.children[0].typeName)
	}

	if spec.children[0].arg != "bar" {
		t.Fatalf("expected arg bar, got %q", spec.children[0].arg)
	}

	if spec.children[0].id != 0 {
		t.Fatalf("expected id 0, got %d", spec.children[0].id)
	}

	if len(spec.children[0].children) != 0 {
		t.Fatalf("expected no children, got %d", len(spec.children[0].children))
	}
}

func TestForkWithChildren(t *testing.T) {
	s := `
	fork[
		literal:all,
		argument:ipv4,
		argument:ipv6,
		argument:string
	]
	`

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	if spec.typeName != "fork" {
		t.Fatalf("expected fork, got %q", spec.typeName)
	}

	if spec.arg != "" {
		t.Fatalf("expected arg '', got %q", spec.arg)
	}

	if spec.id != 0 {
		t.Fatalf("expected id 0, got %d", spec.id)
	}

	if len(spec.children) != 4 {
		t.Fatalf("expected 4 children, got %d", len(spec.children))
	}

	c1 := spec.children[0]
	if c1.typeName != "literal" {
		t.Fatalf("expected literal, got %q", c1.typeName)
	}

	if c1.arg != "all" {
		t.Fatalf("expected arg all, got %q", c1.arg)
	}

	if c1.id != 0 {
		t.Fatalf("expected id 0, got %d", c1.id)
	}

	if len(c1.children) != 0 {
		t.Fatalf("expected no children, got %d", len(c1.children))
	}

	c2 := spec.children[1]
	if c2.typeName != "argument" {
		t.Fatalf("expected argument, got %q", c2.typeName)
	}

	if c2.arg != "ipv4" {
		t.Fatalf("expected arg ipv4, got %q", c2.arg)
	}

	if c2.id != 0 {
		t.Fatalf("expected id 0, got %d", c2.id)
	}

	if len(c2.children) != 0 {
		t.Fatalf("expected no children, got %d", len(c2.children))
	}

	c3 := spec.children[2]
	if c3.typeName != "argument" {
		t.Fatalf("expected argument, got %q", c3.typeName)
	}

	if c3.arg != "ipv6" {
		t.Fatalf("expected arg ipv6, got %q", c3.arg)
	}

	if c3.id != 0 {
		t.Fatalf("expected id 0, got %d", c3.id)
	}

	if len(c3.children) != 0 {
		t.Fatalf("expected no children, got %d", len(c3.children))
	}

	c4 := spec.children[3]
	if c4.typeName != "argument" {
		t.Fatalf("expected argument, got %q", c4.typeName)
	}

	if c4.arg != "string" {
		t.Fatalf("expected arg string, got %q", c4.arg)
	}

	if c4.id != 0 {
		t.Fatalf("expected id 0, got %d", c4.id)
	}

	if len(c4.children) != 0 {
		t.Fatalf("expected no children, got %d", len(c4.children))
	}
}

func TestForkJoinWithID(t *testing.T) {
	s := `
	fork[
		literal:all[
			join.1
		],
		argument:ipv4[
			join.1
		]
	]
	`

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	if spec.typeName != "fork" {
		t.Fatalf("expected fork, got %q", spec.typeName)
	}

	if spec.arg != "" {
		t.Fatalf("expected arg '', got %q", spec.arg)
	}

	if spec.id != 0 {
		t.Fatalf("expected id 0, got %d", spec.id)
	}

	if len(spec.children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(spec.children))
	}

	c1 := spec.children[0]
	if c1.typeName != "literal" {
		t.Fatalf("expected literal, got %q", c1.typeName)
	}

	if c1.arg != "all" {
		t.Fatalf("expected arg all, got %q", c1.arg)
	}

	if c1.id != 0 {
		t.Fatalf("expected id 0, got %d", c1.id)
	}

	if len(c1.children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(c1.children))
	}

	if c1.children[0].typeName != "join" {
		t.Fatalf("expected join, got %q", c1.children[0].typeName)
	}

	if c1.children[0].arg != "" {
		t.Fatalf("expected arg '', got %q", c1.children[0].arg)
	}

	if c1.children[0].id != 1 {
		t.Fatalf("expected id 1, got %d", c1.children[0].id)
	}

	if len(c1.children[0].children) != 0 {
		t.Fatalf("expected no children, got %d", len(c1.children[0].children))
	}

	c2 := spec.children[1]
	if c2.typeName != "argument" {
		t.Fatalf("expected argument, got %q", c2.typeName)
	}

	if c2.arg != "ipv4" {
		t.Fatalf("expected arg ipv4, got %q", c2.arg)
	}

	if c2.id != 0 {
		t.Fatalf("expected id 0, got %d", c2.id)
	}

	if len(c2.children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(c2.children))
	}

	if c2.children[0].typeName != "join" {
		t.Fatalf("expected join, got %q", c2.children[0].typeName)
	}

	if c2.children[0].arg != "" {
		t.Fatalf("expected arg '', got %q", c2.children[0].arg)
	}

	if c2.children[0].id != 1 {
		t.Fatalf("expected id 1, got %d", c2.children[0].id)
	}

	if len(c2.children[0].children) != 0 {
		t.Fatalf("expected no children, got %d", len(c2.children[0].children))
	}
}

func TestInvalidZeroID(t *testing.T) {
	_, err := parseSpec("join.0")
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "invalid id 0") {
		t.Fatalf("expected error to contain 'invalid id 0', got %q", err.Error())
	}
}

func TestDescription(t *testing.T) {
	s, err := parseSpec("literal:foo?\"does the foo\"")
	if err != nil {
		t.Fatal(err)
	}

	if s == nil {
		t.Fatal("expected spec")
	}

	if s.typeName != "literal" {
		t.Fatalf("expected literal, got %q", s.typeName)
	}

	if s.arg != "foo" {
		t.Fatalf("expected arg foo, got %q", s.arg)
	}

	if s.id != 0 {
		t.Fatalf("expected id 0, got %d", s.id)
	}

	if s.description != "does the foo" {
		t.Fatalf("expected description 'does the foo', got %q", s.description)
	}

	if len(s.children) != 0 {
		t.Fatalf("expected no children, got %d", len(s.children))
	}
}

func TestMoreComplicatedDescription(t *testing.T) {
	s := `
	literal:foo.1?"is foo"[
		literal:bar.2?"is bar"
	]	
	`

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	if spec.typeName != "literal" {
		t.Fatalf("expected literal, got %q", spec.typeName)
	}

	if spec.arg != "foo" {
		t.Fatalf("expected arg foo, got %q", spec.arg)
	}

	if spec.id != 1 {
		t.Fatalf("expected id 1, got %d", spec.id)
	}

	if spec.description != "is foo" {
		t.Fatalf("expected description 'is foo', got %q", spec.description)
	}

	if len(spec.children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(spec.children))
	}

	c1 := spec.children[0]
	if c1.typeName != "literal" {
		t.Fatalf("expected literal, got %q", c1.typeName)
	}

	if c1.arg != "bar" {
		t.Fatalf("expected arg bar, got %q", c1.arg)
	}

	if c1.id != 2 {
		t.Fatalf("expected id 2, got %d", c1.id)
	}

	if c1.description != "is bar" {
		t.Fatalf("expected description 'is bar', got %q", c1.description)
	}

	if len(c1.children) != 0 {
		t.Fatalf("expected no children, got %d", len(c1.children))
	}
}
