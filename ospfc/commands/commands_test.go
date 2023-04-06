package commands

import (
	"net/netip"
	"reflect"
	"testing"
)

func TestParseCommand(t *testing.T) {
	s := "show version"
	spec := `
		literal:show[literal:version]
	`
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	AssertMatchesSpec(t, spec, cmd)
}

func TestParseCommandWithParam(t *testing.T) {
	s := "show bgp neighbors A.B.C.D"
	spec := `
		literal:show[literal:bgp[literal:neighbors[param:ipv4]]]
	`

	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	AssertMatchesSpec(t, spec, cmd)
}

func TestParseCommandWithChoice(t *testing.T) {
	s := "show bgp neighbors <A.B.C.D|X:X:X::X|all>"
	spec := `
		literal:show[
			literal:bgp[
				literal:neighbors[
					choice[
						param:ipv4,
						param:ipv6,
						literal:all
					]
				]
			]
		]
	`

	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	AssertMatchesSpec(t, spec, cmd)
}

func TestParseCommandWithChoiceAndTrailingLiteral(t *testing.T) {
	s := "show bgp neighbors <A.B.C.D|X:X:X::X|all> detail"
	spec := `
		literal:show[
			literal:bgp[
				literal:neighbors[
					choice[
						param:ipv4[literal:detail.1],
						param:ipv6[literal:detail.1],
						literal:all[literal:detail.1],
					]
				]
			]
		]
	`

	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	AssertMatchesSpec(t, spec, cmd)
}

func TestMatchLiteral(t *testing.T) {
	s := "show"
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	cmd.handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("show")
	if matches == nil {
		t.Fatal("expected match")
	}

	if len(matches) != 1 {
		t.Fatal("expected 1 match")
	}

	match := matches[0]

	if match.input != "show" {
		t.Fatal("expected input to be 'show'")
	}

	if match.node != cmd {
		t.Fatal("expected node to be cmd")
	}

	if match.addr.IsValid() {
		t.Fatal("expected addr to be unset")
	}

	if match.next != nil {
		t.Fatal("expected next to be nil")
	}
}

func TestMatchLiteralNoHandler(t *testing.T) {
	s := "show"
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("show")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchLiteralPrefix(t *testing.T) {
	s := "show"
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	cmd.handlerFunc = reflect.ValueOf(func() {})
	matches := cmd.Match("sh")
	if len(matches) != 1 {
		t.Fatal("expected 1 match")
	}

	match := matches[0]

	if match.input != "sh" {
		t.Fatal("expected input to be 'sh'")
	}

	if match.node != cmd {
		t.Fatal("expected node to be cmd")
	}

	if match.addr.IsValid() {
		t.Fatal("expected addr to be unset")
	}

	if match.next != nil {
		t.Fatal("expected next to be nil")
	}
}

func TestMatchLiteralInvalid(t *testing.T) {
	s := "show"
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	cmd.handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("sha")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchString(t *testing.T) {
	s := "WORD"
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	cmd.handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("foobar")
	if len(matches) != 1 {
		t.Fatal("expected match")
	}

	match := matches[0]

	if match.input != "foobar" {
		t.Fatal("expected input to be 'foobar'")
	}

	if match.node != cmd {
		t.Fatal("expected node to be cmd")
	}

	if match.addr.IsValid() {
		t.Fatal("expected addr to be unset")
	}

	if match.next != nil {
		t.Fatal("expected next to be nil")
	}
}

func TestMatchStringNoHandler(t *testing.T) {
	s := "WORD"
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("foobar")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchIPv4(t *testing.T) {
	s := "A.B.C.D"
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	cmd.handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("192.168.0.1")
	if len(matches) != 1 {
		t.Fatal("expected match")
	}

	match := matches[0]

	if match.input != "192.168.0.1" {
		t.Fatal("expected input to be '192.168.0.1'")
	}

	if match.node != cmd {
		t.Fatal("expected node to be cmd")
	}

	if match.addr.Compare(netip.MustParseAddr("192.168.0.1")) != 0 {
		t.Fatal("expected addr to be '192.168.0.1'")
	}
}

func TestMatchIPv4NoHandler(t *testing.T) {
	s := "A.B.C.D"
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("192.168.0.1")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchIPv4IncompleteAddr(t *testing.T) {
	s := "A.B.C.D"
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	cmd.handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("192.168.0")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchIPv4InvalidAddr(t *testing.T) {
	s := "A.B.C.D"
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	cmd.handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("300.0.0.1")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchIPv4Nonsense(t *testing.T) {
	s := "A.B.C.D"
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	cmd.handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("foobar")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchIPv4NoMatchIPv6(t *testing.T) {
	s := "A.B.C.D"
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	cmd.handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("2001:db8::68")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchIPv6(t *testing.T) {
	s := "X:X:X::X"
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	cmd.handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("2001:db8::68")
	if len(matches) != 1 {
		t.Fatal("expected match")
	}

	match := matches[0]

	if match.input != "2001:db8::68" {
		t.Fatal("expected input to be '2001:db8::68'")
	}

	if match.node != cmd {
		t.Fatal("expected node to be cmd")
	}

	if match.addr.Compare(netip.MustParseAddr("2001:db8::68")) != 0 {
		t.Fatal("expected addr to be '2001:db8::68'")
	}

	if match.next != nil {
		t.Fatal("expected next to be nil")
	}
}

func TestMatchIPv6NoHandler(t *testing.T) {
	s := "X:X:X::X"
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("2001:db8::68")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchIPv6IncompleteAddr(t *testing.T) {
	s := "X:X:X::X"
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	cmd.handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("2001:db8")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchIPv6InvalidAddr(t *testing.T) {
	s := "X:X:X::X"
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	cmd.handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("2001:db8::g")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchIPv6Nonsense(t *testing.T) {
	s := "X:X:X::X"
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	cmd.handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("foobar")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchIPv6NoMatchIPv4(t *testing.T) {
	s := "X:X:X::X"
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	cmd.handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("192.168.0.1")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchIPv6MappedIPv4(t *testing.T) {
	s := "X:X:X::X"
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	cmd.handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("::ffff:192.168.0.1")
	if len(matches) != 1 {
		t.Fatal("expected match")
	}

	match := matches[0]

	if match.input != "::ffff:192.168.0.1" {
		t.Fatal("expected input to be '::ffff:192.168.0.1")
	}

	if match.node != cmd {
		t.Fatal("expected node to be cmd")
	}

	if match.addr.Compare(netip.MustParseAddr("::ffff:192.168.0.1")) != 0 {
		t.Fatal("expected addr to be '::ffff:192.168.0.1'")
	}

	if match.next != nil {
		t.Fatal("expected next to be nil")
	}
}

func TestMatchChoiceLiteral(t *testing.T) {
	s := "<foo|bar>"
	cmd, err := parseCommand(s)
	if err != nil {
		t.Fatal(err)
	}

	cmd.children[0].handlerFunc = reflect.ValueOf(func() {})
	cmd.children[1].handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("foo")
	if len(matches) != 1 {
		t.Fatal("expected match")
	}

	match := matches[0]

	if match.input != "foo" {
		t.Fatal("expected input to be 'foo'")
	}

	if match.node != cmd.children[0] {
		t.Fatal("expected node to be cmd.Children()[0]")
	}

	if match.addr.IsValid() {
		t.Fatal("expected addr to be unset")
	}

	if match.next != nil {
		t.Fatal("expected next to be nil")
	}

	matches = cmd.Match("bar")

	if len(matches) != 1 {
		t.Fatal("expected match")
	}

	match = matches[0]

	if match.input != "bar" {
		t.Fatal("expected input to be 'bar'")
	}

	if match.node != cmd.children[1] {
		t.Fatal("expected node to be cmd.Children()[1]")
	}

	if match.addr.IsValid() {
		t.Fatal("expected addr to be unset")
	}

	if match.next != nil {
		t.Fatal("expected next to be nil")
	}

	matches = cmd.Match("baz")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}
