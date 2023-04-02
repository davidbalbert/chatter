package commands

import (
	"errors"
	"net/netip"
	"reflect"
	"strings"
	"testing"
)

func TestParseFork(t *testing.T) {
	spec, err := ParseSpec("fork")
	if err != nil {
		t.Fatal(err)
	}

	if spec.typeName != "fork" {
		t.Fatalf("expected fork, got %q", spec.typeName)
	}

	if spec.value != "" {
		t.Fatalf("expected empty value, got %q", spec.value)
	}

	if len(spec.children) != 0 {
		t.Fatalf("expected no children, got %d", len(spec.children))
	}

	if spec.id != 0 {
		t.Fatalf("expected id 0, got %d", spec.id)
	}
}

func TestParseJoin(t *testing.T) {
	spec, err := ParseSpec("join")
	if err != nil {
		t.Fatal(err)
	}

	if spec.typeName != "join" {
		t.Fatalf("expected join, got %q", spec.typeName)
	}

	if spec.value != "" {
		t.Fatalf("expected empty value, got %q", spec.value)
	}

	if len(spec.children) != 0 {
		t.Fatalf("expected no children, got %d", len(spec.children))
	}

	if spec.id != 0 {
		t.Fatalf("expected id 0, got %d", spec.id)
	}
}

func TestParseJoinWithID(t *testing.T) {
	spec, err := ParseSpec("join.1")
	if err != nil {
		t.Fatal(err)
	}

	if spec.typeName != "join" {
		t.Fatalf("expected join, got %q", spec.typeName)
	}

	if spec.value != "" {
		t.Fatalf("expected empty value, got %q", spec.value)
	}

	if len(spec.children) != 0 {
		t.Fatalf("expected no children, got %d", len(spec.children))
	}

	if spec.id != 1 {
		t.Fatalf("expected id 1, got %d", spec.id)
	}
}

func TestParseLiteral(t *testing.T) {
	spec, err := ParseSpec("literal:foo")
	if err != nil {
		t.Fatal(err)
	}

	if spec.typeName != "literal" {
		t.Fatalf("expected literal, got %q", spec.typeName)
	}

	if spec.value != "foo" {
		t.Fatalf("expected arg foo, got %q", spec.value)
	}

	if len(spec.children) != 0 {
		t.Fatalf("expected no children, got %d", len(spec.children))
	}

	if spec.id != 0 {
		t.Fatalf("expected id 0, got %d", spec.id)
	}
}

func TestParseLiteralMissingValue(t *testing.T) {
	_, err := ParseSpec("literal")
	if err == nil {
		t.Fatal("expected error")
	}

	_, err = ParseSpec("literal:")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseArgumentString(t *testing.T) {
	spec, err := ParseSpec("argument:string")
	if err != nil {
		t.Fatal(err)
	}

	if spec.typeName != "argument" {
		t.Fatalf("expected argument, got %q", spec.typeName)
	}

	if spec.argType != argumentTypeString {
		t.Fatalf("expected argument type %s, got %s", argumentTypeString, spec.argType)
	}

	if len(spec.children) != 0 {
		t.Fatalf("expected no children, got %d", len(spec.children))
	}

	if spec.id != 0 {
		t.Fatalf("expected id 0, got %d", spec.id)
	}
}

func TestParseArgumentIPv4(t *testing.T) {
	spec, err := ParseSpec("argument:ipv4")
	if err != nil {
		t.Fatal(err)
	}

	if spec.typeName != "argument" {
		t.Fatalf("expected argument, got %q", spec.typeName)
	}

	if spec.argType != argumentTypeIPv4 {
		t.Fatalf("expected argument type %s, got %q", argumentTypeIPv4, spec.argType)
	}

	if len(spec.children) != 0 {
		t.Fatalf("expected no children, got %d", len(spec.children))
	}

	if spec.id != 0 {
		t.Fatalf("expected id 0, got %d", spec.id)
	}
}

func TestParseArgumentIPv6(t *testing.T) {
	spec, err := ParseSpec("argument:ipv6")
	if err != nil {
		t.Fatal(err)
	}

	if spec.typeName != "argument" {
		t.Fatalf("expected argument, got %q", spec.typeName)
	}

	if spec.argType != argumentTypeIPv6 {
		t.Fatalf("expected argument type %s, got %s", argumentTypeIPv6, spec.argType)
	}

	if len(spec.children) != 0 {
		t.Fatalf("expected no children, got %d", len(spec.children))
	}

	if spec.id != 0 {
		t.Fatalf("expected id 0, got %d", spec.id)
	}
}

func TestParseArgumentBadType(t *testing.T) {
	_, err := ParseSpec("argument:foo")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseArgumentMissingType(t *testing.T) {
	_, err := ParseSpec("argument")
	if err == nil {
		t.Fatal("expected error")
	}

	_, err = ParseSpec("argument:")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLiteralWithChild(t *testing.T) {
	s, err := ParseSpec("literal:foo[literal:bar]")
	if err != nil {
		t.Fatal(err)
	}

	if s == nil {
		t.Fatal("expected spec")
	}

	if s.typeName != "literal" {
		t.Fatalf("expected literal, got %q", s.typeName)
	}

	if s.value != "foo" {
		t.Fatalf("expected value foo, got %q", s.value)
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

	if s.children[0].value != "bar" {
		t.Fatalf("expected value bar, got %q", s.children[0].value)
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

	spec, err := ParseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	if spec.typeName != "literal" {
		t.Fatalf("expected literal, got %q", spec.typeName)
	}

	if spec.value != "foo" {
		t.Fatalf("expected value foo, got %q", spec.value)
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

	if spec.children[0].value != "bar" {
		t.Fatalf("expected value bar, got %q", spec.children[0].value)
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
		argument:string,
	]
	`

	spec, err := ParseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	if spec.typeName != "fork" {
		t.Fatalf("expected fork, got %q", spec.typeName)
	}

	if spec.value != "" {
		t.Fatalf("expected value '', got %q", spec.value)
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

	if c1.value != "all" {
		t.Fatalf("expected arg all, got %q", c1.value)
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

	if c2.argType != argumentTypeIPv4 {
		t.Fatalf("expected arg %s, got %s", argumentTypeIPv4, c2.argType)
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

	if c3.argType != argumentTypeIPv6 {
		t.Fatalf("expected value %s, got %s", argumentTypeIPv6, c3.argType)
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

	if c4.argType != argumentTypeString {
		t.Fatalf("expected value %s, got %s", argumentTypeString, c4.argType)
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

	spec, err := ParseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	if spec.typeName != "fork" {
		t.Fatalf("expected fork, got %q", spec.typeName)
	}

	if spec.value != "" {
		t.Fatalf("expected value '', got %q", spec.value)
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

	if c1.value != "all" {
		t.Fatalf("expected value all, got %q", c1.value)
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

	if c1.children[0].value != "" {
		t.Fatalf("expected value '', got %q", c1.children[0].value)
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

	if c2.argType != argumentTypeIPv4 {
		t.Fatalf("expected argType %s, got %s", argumentTypeIPv4, c2.argType)
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

	if c2.children[0].value != "" {
		t.Fatalf("expected value '', got %q", c2.children[0].value)
	}

	if c2.children[0].id != 1 {
		t.Fatalf("expected id 1, got %d", c2.children[0].id)
	}

	if len(c2.children[0].children) != 0 {
		t.Fatalf("expected no children, got %d", len(c2.children[0].children))
	}
}

func TestInvalidZeroID(t *testing.T) {
	_, err := ParseSpec("join.0")
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "invalid id 0") {
		t.Fatalf("expected error to contain 'invalid id 0', got %q", err.Error())
	}
}

func TestDescription(t *testing.T) {
	s, err := ParseSpec("literal:foo?\"does the foo\"")
	if err != nil {
		t.Fatal(err)
	}

	if s == nil {
		t.Fatal("expected spec")
	}

	if s.typeName != "literal" {
		t.Fatalf("expected literal, got %q", s.typeName)
	}

	if s.value != "foo" {
		t.Fatalf("expected arg value, got %q", s.value)
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

	spec, err := ParseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	if spec.typeName != "literal" {
		t.Fatalf("expected literal, got %q", spec.typeName)
	}

	if spec.value != "foo" {
		t.Fatalf("expected value foo, got %q", spec.value)
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

	if c1.value != "bar" {
		t.Fatalf("expected value bar, got %q", c1.value)
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

func TestAutocomplete(t *testing.T) {
	s, err := ParseSpec("argument:ipv4!A?\"Autocompletes IPv4 addresses\"")
	if err != nil {
		t.Fatal(err)
	}

	if s == nil {
		t.Fatal("expected spec")
	}

	if s.typeName != "argument" {
		t.Fatalf("expected argument, got %q", s.typeName)
	}

	if s.argType != argumentTypeIPv4 {
		t.Fatalf("expected argType %s, got %s", argumentTypeIPv4, s.argType)
	}

	if s.id != 0 {
		t.Fatalf("expected id 0, got %d", s.id)
	}

	if s.description != "Autocompletes IPv4 addresses" {
		t.Fatalf("expected description 'Autocompletes IPv4 addresses', got %q", s.description)
	}

	if s.hasAutocomplete != true {
		t.Fatal("expected hasAutocomplete to be true")
	}
}

func TestHandler(t *testing.T) {
	s := `
	literal:show[
		argument:ipv4!A[
			argument:ipv6!A!Hfunc(ipv4, ipv6)?"Has autocomplete and handler"
		]
	]
	`

	spec, err := ParseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	if spec.typeName != "literal" {
		t.Fatalf("expected literal, got %q", spec.typeName)
	}

	if spec.value != "show" {
		t.Fatalf("expected value show, got %q", spec.value)
	}

	if spec.id != 0 {
		t.Fatalf("expected id 0, got %d", spec.id)
	}

	if spec.description != "" {
		t.Fatalf("expected description '', got %q", spec.description)
	}

	if len(spec.children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(spec.children))
	}

	c1 := spec.children[0]
	if c1.typeName != "argument" {
		t.Fatalf("expected argument, got %q", c1.typeName)
	}

	if c1.argType != argumentTypeIPv4 {
		t.Fatalf("expected argType %s, got %s", argumentTypeIPv4, c1.argType)
	}

	if c1.id != 0 {
		t.Fatalf("expected id 0, got %d", c1.id)
	}

	if c1.description != "" {
		t.Fatalf("expected description '', got %q", c1.description)
	}

	if c1.hasAutocomplete != true {
		t.Fatal("expected hasAutocomplete to be true")
	}

	if c1.handler != nil {
		t.Fatal("expected handler to be empty")
	}

	if len(c1.children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(c1.children))
	}

	c2 := c1.children[0]
	if c2.typeName != "argument" {
		t.Fatalf("expected argument, got %q", c2.typeName)
	}

	if c2.argType != argumentTypeIPv6 {
		t.Fatalf("expected arg %s, got %q", argumentTypeIPv6, c2.argType)
	}

	if c2.id != 0 {
		t.Fatalf("expected id 0, got %d", c2.id)
	}

	if c2.description != "Has autocomplete and handler" {
		t.Fatalf("expected description 'Has autocomplete and handler', got %q", c2.description)
	}

	if c2.hasAutocomplete != true {
		t.Fatal("expected hasAutocomplete to be true")
	}

	if c2.handler == nil {
		t.Fatal("expected handler to not be empty")
	}

	in := []reflect.Type{reflect.TypeOf(netip.Addr{}), reflect.TypeOf(netip.Addr{})}
	out := []reflect.Type{reflect.TypeOf(errors.New(""))}
	signature := reflect.FuncOf(in, out, false)

	if *c2.handler != signature {
		t.Fatalf("expected handler to be %v, got %v", signature, c1.handler)
	}
}
