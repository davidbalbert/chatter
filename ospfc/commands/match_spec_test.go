package commands

import (
	"net/netip"
	"reflect"
	"testing"
)

func TestMatchSpecLiteral(t *testing.T) {
	s := "foo bar baz"

	actual, err := parseMatchSpecs(s)
	if err != nil {
		t.Fatal(err)
	}

	expected := &matchSpecs{
		[]*matchSpec{
			{[]*matchSpecPart{{t: ntLiteral, s: "foo"}, {t: ntLiteral, s: "bar"}, {t: ntLiteral, s: "baz"}}},
		},
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func TestMatchSpecMulti(t *testing.T) {
	s := `
		foo bar
		baz qux
	`

	actual, err := parseMatchSpecs(s)
	if err != nil {
		t.Fatal(err)
	}

	expected := &matchSpecs{
		[]*matchSpec{
			{[]*matchSpecPart{{t: ntLiteral, s: "foo"}, {t: ntLiteral, s: "bar"}}},
			{[]*matchSpecPart{{t: ntLiteral, s: "baz"}, {t: ntLiteral, s: "qux"}}},
		},
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func TestMatchSpecParamIPv4(t *testing.T) {
	s := "show ip route ipv4:1.2.3.4"

	actual, err := parseMatchSpecs(s)
	if err != nil {
		t.Fatal(err)
	}

	expected := &matchSpecs{
		[]*matchSpec{
			{[]*matchSpecPart{
				{t: ntLiteral, s: "show"},
				{t: ntLiteral, s: "ip"},
				{t: ntLiteral, s: "route"},
				{t: ntParamIPv4, s: "1.2.3.4", addr: netip.MustParseAddr("1.2.3.4")},
			}},
		},
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func TestMatchSpecInvalidIPv4(t *testing.T) {
	s := "show ip route ipv4:300.2.3.4"

	_, err := parseMatchSpecs(s)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMatchSpecNonsenseIPv4(t *testing.T) {
	s := "show ip route ipv4:foo"

	_, err := parseMatchSpecs(s)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMatchSpecParamIPv6(t *testing.T) {
	s := "show ip route ipv6:2001:db8::1"

	actual, err := parseMatchSpecs(s)
	if err != nil {
		t.Fatal(err)
	}

	expected := &matchSpecs{
		[]*matchSpec{
			{[]*matchSpecPart{
				{t: ntLiteral, s: "show"},
				{t: ntLiteral, s: "ip"},
				{t: ntLiteral, s: "route"},
				{t: ntParamIPv6, s: "2001:db8::1", addr: netip.MustParseAddr("2001:db8::1")},
			}},
		},
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func TestMatchSpecParamIPv6MappedIPv4(t *testing.T) {
	s := "show ip route ipv6:::ffff:1.2.3.4"

	actual, err := parseMatchSpecs(s)
	if err != nil {
		t.Fatal(err)
	}

	expected := &matchSpecs{
		[]*matchSpec{
			{[]*matchSpecPart{
				{t: ntLiteral, s: "show"},
				{t: ntLiteral, s: "ip"},
				{t: ntLiteral, s: "route"},
				{t: ntParamIPv6, s: "::ffff:1.2.3.4", addr: netip.MustParseAddr("::ffff:1.2.3.4")},
			}},
		},
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func TestMatchSpecInvalidIPv6(t *testing.T) {
	s := "show ip route ipv6:2001:db8"

	_, err := parseMatchSpecs(s)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMatchSpecNonsenseIPv6(t *testing.T) {
	s := "show ip route ipv6:foo"

	_, err := parseMatchSpecs(s)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMatchSpecParamString(t *testing.T) {
	s := "show ip route string:foo"

	actual, err := parseMatchSpecs(s)
	if err != nil {
		t.Fatal(err)
	}

	expected := &matchSpecs{
		[]*matchSpec{
			{[]*matchSpecPart{
				{t: ntLiteral, s: "show"},
				{t: ntLiteral, s: "ip"},
				{t: ntLiteral, s: "route"},
				{t: ntParamString, s: "foo"},
			}},
		},
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func TestMatchSpecMatchLiterals(t *testing.T) {
	cmd, err := parseCommand("show ip route")
	if err != nil {
		t.Fatal(err)
	}

	cmd.children[0].children[0].handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("show ip route")

	specs, err := parseMatchSpecs("show ip route")
	if err != nil {
		t.Fatal(err)
	}

	err = specs.match(matches)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMatchSpecNoHandler(t *testing.T) {
	cmd, err := parseCommand("show ip route")
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("show ip route")

	specs, err := parseMatchSpecs("show ip route")
	if err != nil {
		t.Fatal(err)
	}

	err = specs.match(matches)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMatchSpecMatchParams(t *testing.T) {
	cmd, err := parseCommand("show ip route A.B.C.D")
	if err != nil {
		t.Fatal(err)
	}

	cmd.children[0].children[0].children[0].handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("show ip route 1.2.3.4")

	specs, err := parseMatchSpecs("show ip route ipv4:1.2.3.4")
	if err != nil {
		t.Fatal(err)
	}

	err = specs.match(matches)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMatchSpecMatchParamsIPv6(t *testing.T) {
	cmd, err := parseCommand("show ip route X:X:X::X")
	if err != nil {
		t.Fatal(err)
	}

	cmd.children[0].children[0].children[0].handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("show ip route 2001:db8::1")

	specs, err := parseMatchSpecs("show ip route ipv6:2001:db8::1")
	if err != nil {
		t.Fatal(err)
	}

	err = specs.match(matches)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMatchSpecAmbiguousMatch(t *testing.T) {
	cmd, err := parseCommand("show <ip|interface>")
	if err != nil {
		t.Fatal(err)
	}

	cmd.children[0].children[0].handlerFunc = reflect.ValueOf(func() {})
	cmd.children[0].children[1].handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("sh i")

	s := `
		show ip
		show interface
	`

	specs, err := parseMatchSpecs(s)
	if err != nil {
		t.Fatal(err)
	}

	err = specs.match(matches)
	if err != nil {
		t.Fatal(err)
	}
}
