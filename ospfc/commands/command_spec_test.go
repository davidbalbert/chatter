package commands

import (
	"net/netip"
	"reflect"
	"strings"
	"testing"
)

func TestParseChoice(t *testing.T) {
	spec, err := parseSpec("choice")
	if err != nil {
		t.Fatal(err)
	}

	if spec.t != ntChoice {
		t.Fatalf("expected choice, got %q", spec.t)
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

func TestParseLiteral(t *testing.T) {
	spec, err := parseSpec("literal:foo")
	if err != nil {
		t.Fatal(err)
	}

	if spec.t != ntLiteral {
		t.Fatalf("expected literal, got %q", spec.t)
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

func TestParseLiteralWithID(t *testing.T) {
	spec, err := parseSpec("literal:foo.1")
	if err != nil {
		t.Fatal(err)
	}

	if spec.t != ntLiteral {
		t.Fatalf("expected literal, got %q", spec.t)
	}

	if spec.value != "foo" {
		t.Fatalf("expected value foo, got %q", spec.value)
	}

	if len(spec.children) != 0 {
		t.Fatalf("expected no children, got %d", len(spec.children))
	}

	if spec.id != 1 {
		t.Fatalf("expected id 1, got %d", spec.id)
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
	spec, err := parseSpec("param:string")
	if err != nil {
		t.Fatal(err)
	}

	if spec.t != ntParamString {
		t.Fatalf("expected param:string, got %q", spec.t)
	}

	if len(spec.children) != 0 {
		t.Fatalf("expected no children, got %d", len(spec.children))
	}

	if spec.id != 0 {
		t.Fatalf("expected id 0, got %d", spec.id)
	}
}

func TestParseArgumentIPv4(t *testing.T) {
	spec, err := parseSpec("param:ipv4")
	if err != nil {
		t.Fatal(err)
	}

	if spec.t != ntParamIPv4 {
		t.Fatalf("expected param:ipv4, got %q", spec.t)
	}

	if len(spec.children) != 0 {
		t.Fatalf("expected no children, got %d", len(spec.children))
	}

	if spec.id != 0 {
		t.Fatalf("expected id 0, got %d", spec.id)
	}
}

func TestParseArgumentIPv6(t *testing.T) {
	spec, err := parseSpec("param:ipv6")
	if err != nil {
		t.Fatal(err)
	}

	if spec.t != ntParamIPv6 {
		t.Fatalf("expected param:ipv6, got %q", spec.t)
	}

	if len(spec.children) != 0 {
		t.Fatalf("expected no children, got %d", len(spec.children))
	}

	if spec.id != 0 {
		t.Fatalf("expected id 0, got %d", spec.id)
	}
}

func TestParseArgumentBadType(t *testing.T) {
	_, err := parseSpec("param:foo")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseArgumentMissingType(t *testing.T) {
	_, err := parseSpec("param")
	if err == nil {
		t.Fatal("expected error")
	}

	_, err = parseSpec("param:")
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

	if s.t != ntLiteral {
		t.Fatalf("expected literal, got %q", s.t)
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

	if s.children[0].t != ntLiteral {
		t.Fatalf("expected literal, got %q", s.children[0].t)
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

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	if spec.t != ntLiteral {
		t.Fatalf("expected literal, got %q", spec.t)
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

	if spec.children[0].t != ntLiteral {
		t.Fatalf("expected literal, got %q", spec.children[0].t)
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
	choice[
		literal:all,
		param:ipv4,
		param:ipv6,
		param:string,
	]
	`

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	if spec.t != ntChoice {
		t.Fatalf("expected fork, got %q", spec.t)
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
	if c1.t != ntLiteral {
		t.Fatalf("expected literal, got %q", c1.t)
	}

	if c1.value != "all" {
		t.Fatalf("expected value all, got %q", c1.value)
	}

	if c1.id != 0 {
		t.Fatalf("expected id 0, got %d", c1.id)
	}

	if len(c1.children) != 0 {
		t.Fatalf("expected no children, got %d", len(c1.children))
	}

	c2 := spec.children[1]
	if c2.t != ntParamIPv4 {
		t.Fatalf("expected param:ipv4, got %q", c2.t)
	}

	if c2.id != 0 {
		t.Fatalf("expected id 0, got %d", c2.id)
	}

	if len(c2.children) != 0 {
		t.Fatalf("expected no children, got %d", len(c2.children))
	}

	c3 := spec.children[2]
	if c3.t != ntParamIPv6 {
		t.Fatalf("expected param:ipv6, got %q", c3.t)
	}

	if c3.id != 0 {
		t.Fatalf("expected id 0, got %d", c3.id)
	}

	if len(c3.children) != 0 {
		t.Fatalf("expected no children, got %d", len(c3.children))
	}

	c4 := spec.children[3]
	if c4.t != ntParamString {
		t.Fatalf("expected param:string, got %q", c4.t)
	}

	if c4.id != 0 {
		t.Fatalf("expected id 0, got %d", c4.id)
	}

	if len(c4.children) != 0 {
		t.Fatalf("expected no children, got %d", len(c4.children))
	}
}

func TestInvalidZeroID(t *testing.T) {
	_, err := parseSpec("choice.0")
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

	if s.t != ntLiteral {
		t.Fatalf("expected literal, got %q", s.t)
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

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	if spec.t != ntLiteral {
		t.Fatalf("expected literal, got %q", spec.t)
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
	if c1.t != ntLiteral {
		t.Fatalf("expected literal, got %q", c1.t)
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
	s, err := parseSpec("param:ipv4!A?\"Autocompletes IPv4 addresses\"")
	if err != nil {
		t.Fatal(err)
	}

	if s == nil {
		t.Fatal("expected spec")
	}

	if s.t != ntParamIPv4 {
		t.Fatalf("expected param:ipv4, got %q", s.t)
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
		param:ipv4!A[
			param:ipv6!A!Hfunc(addr, addr)?"Has autocomplete and handler"
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

	if spec.t != ntLiteral {
		t.Fatalf("expected literal, got %q", spec.t)
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
	if c1.t != ntParamIPv4 {
		t.Fatalf("expected param:ipv4, got %q", c1.t)
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
	if c2.t != ntParamIPv6 {
		t.Fatalf("expected param:ipv6, got %q", c2.t)
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

	signature := reflect.TypeOf(func(netip.Addr, netip.Addr) error { return nil })

	if *c2.handler != signature {
		t.Fatalf("expected handler to be %v, got %v", signature, *c2.handler)
	}
}

func TestEmptyHandler(t *testing.T) {
	s := `literal:show!Hfunc()?"Has empty handler"`

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	if spec.t != ntLiteral {
		t.Fatalf("expected literal, got %q", spec.t)
	}

	if spec.value != "show" {
		t.Fatalf("expected value show, got %q", spec.value)
	}

	if spec.id != 0 {
		t.Fatalf("expected id 0, got %d", spec.id)
	}

	if spec.description != "Has empty handler" {
		t.Fatalf("expected description 'Has empty handler', got %q", spec.description)
	}

	if spec.hasAutocomplete != false {
		t.Fatal("expected hasAutocomplete to be false")
	}

	if spec.handler == nil {
		t.Fatal("expected handler to not be empty")
	}

	if *spec.handler != reflect.TypeOf(func() error { return nil }) {
		t.Fatalf("expected handler to be %v, got %v", reflect.TypeOf(func() error { return nil }), *spec.handler)
	}
}

func TestMatcher(t *testing.T) {
	s := `
	literal:show[
		literal:version
	]`

	g := &Node{
		t:        ntLiteral,
		value:    "show",
		children: []*Node{{t: ntLiteral, value: "version"}},
	}

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	m := newCommandSpecMatcher()
	err = m.match(spec.pathComponent(), g, spec)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMatcherError(t *testing.T) {
	s := `
	literal:show[
		literal:name
	]`

	g := &Node{
		t:        ntLiteral,
		value:    "show",
		children: []*Node{{t: ntLiteral, value: "version"}},
	}

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	m := newCommandSpecMatcher()
	err = m.match("/"+spec.pathComponent(), g, spec)
	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "/literal:show/literal:name: expected literal:name, got literal:version" {
		t.Fatalf("expected error to be '/literal:show/literal:name: expected literal:name, got literal:version', got %q", err.Error())
	}
}

func TestMatcherErrorDifferentType(t *testing.T) {
	s := `
	literal:show[
		param:ipv4
	]`

	g := &Node{
		t:        ntLiteral,
		value:    "show",
		children: []*Node{{t: ntLiteral, value: "version"}},
	}

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	m := newCommandSpecMatcher()
	err = m.match("/"+spec.pathComponent(), g, spec)
	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "/literal:show/param:ipv4: expected type param:ipv4, got literal" {
		t.Fatalf("expected error to be '/literal:show/param:ipv4: expected type param:ipv4, got literal', got %q", err.Error())
	}
}

func TestMatcherErrorDifferentArgType(t *testing.T) {
	s := `
	literal:show[
		param:ipv4
	]`

	g := &Node{
		t:        ntLiteral,
		value:    "show",
		children: []*Node{{t: ntParamIPv6}},
	}

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	m := newCommandSpecMatcher()
	err = m.match("/"+spec.pathComponent(), g, spec)
	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "/literal:show/param:ipv4: expected type param:ipv4, got param:ipv6" {
		t.Fatalf("expected error to be '/literal:show/argument:ipv4: expected type param:ipv4, got param:ipv6', got %q", err.Error())
	}
}

func TestMatcherErrorNoChildren(t *testing.T) {
	s := `
	literal:show
	`

	g := &Node{
		t:        ntLiteral,
		value:    "show",
		children: []*Node{{t: ntLiteral, value: "version"}},
	}

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	m := newCommandSpecMatcher()
	err = m.match("/"+spec.pathComponent(), g, spec)
	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "/literal:show: expected 0 children, got 1" {
		t.Fatalf("expected error to be '/literal:show: expected 0 children, got 1', got %q", err.Error())
	}
}

func TestMatcherErrorTooManyChildren(t *testing.T) {
	s := `
	choice[
		literal:foo,
		literal:bar
	]`

	g := &Node{
		t: ntChoice,
		children: []*Node{
			{t: ntLiteral, value: "foo"},
			{t: ntLiteral, value: "bar"},
			{t: ntLiteral, value: "baz"},
		},
	}

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	m := newCommandSpecMatcher()
	err = m.match("/"+spec.pathComponent(), g, spec)
	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "/choice: expected 2 children, got 3" {
		t.Fatalf("expected error to be '/choice: expected 2 children, got 3', got %q", err.Error())
	}
}

func TestMatcherErrorMissingChild(t *testing.T) {
	s := `
	choice[
		literal:foo,
		literal:bar
	]`

	g := &Node{
		t:        ntChoice,
		children: []*Node{{t: ntLiteral, value: "foo"}},
	}

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	m := newCommandSpecMatcher()
	err = m.match("/"+spec.pathComponent(), g, spec)
	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "/choice: expected 2 children, got 1" {
		t.Fatalf("expected error to be '/choice: expected 2 children, got 1', got %q", err.Error())
	}
}

// TODO: go maps don't guarantee order. Either make Node.Children() return children in a guaranteed order,
// remove this test, or make specs stored in a map as well.
func TestMatcherErrorChildOrder(t *testing.T) {
	s := `
	choice[
		literal:foo,
		literal:bar
	]`

	g := &Node{
		t: ntChoice,
		children: []*Node{
			{t: ntLiteral, value: "bar"},
			{t: ntLiteral, value: "foo"},
		}}

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	m := newCommandSpecMatcher()
	err = m.match("/"+spec.pathComponent(), g, spec)
	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "/choice/literal:foo: expected literal:foo, got literal:bar" {
		t.Fatalf("expected error to be '/choice/literal:foo: expected literal \"foo\", got \"bar\"', got %q", err.Error())
	}
}

func TestMatcherReferenceIdentity(t *testing.T) {
	s := `
	choice[
		literal:foo[
			literal:baz.1
		],
		literal:bar[
			literal:baz.1
		]
	]`

	baz := &Node{
		t:     ntLiteral,
		value: "baz",
	}

	g := &Node{
		t: ntChoice,
		children: []*Node{
			{t: ntLiteral, value: "foo", children: []*Node{baz}},
			{t: ntLiteral, value: "bar", children: []*Node{baz}},
		},
	}

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	m := newCommandSpecMatcher()
	err = m.match("/"+spec.pathComponent(), g, spec)
	if err != nil {
		t.Fatal(err)
	}

	ref, ok := m.references["literal:baz.1"]
	if !ok {
		t.Fatal("expected baz to be referenced")
	}

	if ref != baz {
		t.Fatal("expected jref to be the same as j")
	}
}

func TestMatcherReferenceIdentitySeparate(t *testing.T) {
	s := `
	choice[
		literal:foo[
			literal:baz.1
		],
		literal:bar[
			literal:baz.2
		]
	]`

	baz1 := &Node{
		t:     ntLiteral,
		value: "baz",
	}

	baz2 := &Node{
		t:     ntLiteral,
		value: "baz",
	}

	g := &Node{
		t: ntChoice,
		children: []*Node{
			{t: ntLiteral, value: "foo", children: []*Node{baz1}},
			{t: ntLiteral, value: "bar", children: []*Node{baz2}},
		},
	}

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	m := newCommandSpecMatcher()
	err = m.match("/"+spec.pathComponent(), g, spec)
	if err != nil {
		t.Fatal(err)
	}

	ref, ok := m.references["literal:baz.1"]
	if !ok {
		t.Fatal("expected join to be referenced")
	}

	if ref != baz1 {
		t.Fatal("expected ref to be the same as baz1")
	}

	ref, ok = m.references["literal:baz.2"]
	if !ok {
		t.Fatal("expected join to be referenced")
	}

	if ref != baz2 {
		t.Fatal("expected ref to be the same as baz2")
	}

	if baz1 == baz2 {
		t.Fatal("expected baz1 and baz2 to be different")
	}
}

func TestReferenceIdentitySeparateIncorrect(t *testing.T) {
	s := `
		choice[
			literal:foo.1,
			literal:foo.2,
		]
	`

	foo1 := &Node{
		t:     ntLiteral,
		value: "foo",
	}

	c := &Node{
		t: ntChoice,
		children: []*Node{
			foo1,
			foo1,
		},
	}

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	m := newCommandSpecMatcher()
	err = m.match("/"+spec.pathComponent(), c, spec)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMatcherReferenceIdentitySkipsOtherAssertions(t *testing.T) {
	s := `
	choice[
		literal:foo[
			literal:baz.1[literal:qux]
		],
		literal:bar[
			literal:baz.1
		]
	]`

	baz := &Node{
		t:        ntLiteral,
		value:    "baz",
		children: []*Node{{t: ntLiteral, value: "qux"}},
	}

	g := &Node{
		t: ntChoice,
		children: []*Node{
			{t: ntLiteral, value: "foo", children: []*Node{baz}},
			{t: ntLiteral, value: "bar", children: []*Node{baz}},
		},
	}

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	if spec == nil {
		t.Fatal("expected spec")
	}

	m := newCommandSpecMatcher()
	err = m.match("/"+spec.pathComponent(), g, spec)
	if err != nil {
		t.Fatal(err)
	}

	ref, ok := m.references["literal:baz.1"]
	if !ok {
		t.Fatal("expected baz to be referenced")
	}

	if ref != baz {
		t.Fatal("expected ref to be the same as baz")
	}
}
