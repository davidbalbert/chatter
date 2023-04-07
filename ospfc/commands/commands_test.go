package commands

import (
	"net/netip"
	"reflect"
	"strings"
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

	AssertMatchesCommandSpec(t, spec, cmd)
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

	AssertMatchesCommandSpec(t, spec, cmd)
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

	AssertMatchesCommandSpec(t, spec, cmd)
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

	AssertMatchesCommandSpec(t, spec, cmd)
}

func TestMergeDescription(t *testing.T) {
	spec1 := `
		literal:show[literal:version]
	`

	cmd1, err := parseCommand("show version")
	if err != nil {
		t.Fatal(err)
	}

	AssertMatchesCommandSpec(t, spec1, cmd1)

	spec2 := `
		literal:show[literal:version?"Show version information"]
	`

	cmd2, err := parseCommand("show version")
	if err != nil {
		t.Fatal(err)
	}

	cmd2.children[0].description = "Show version information"

	AssertMatchesCommandSpec(t, spec2, cmd2)

	spec3 := `
		literal:show[literal:version?"Show version information"]
	`

	cmd3, err := cmd1.Merge(cmd2)
	if err != nil {
		t.Fatal(err)
	}
	AssertMatchesCommandSpec(t, spec3, cmd3)
}

func TestMergeHandler(t *testing.T) {
	spec1 := `
		literal:show[literal:version]
	`

	cmd1, err := parseCommand("show version")
	if err != nil {
		t.Fatal(err)
	}

	AssertMatchesCommandSpec(t, spec1, cmd1)

	spec2 := `
		literal:show[literal:version!Hfunc()]
	`

	cmd2, err := parseCommand("show version")
	if err != nil {
		t.Fatal(err)
	}

	cmd2.children[0].handlerFunc = reflect.ValueOf(func() error { return nil })

	AssertMatchesCommandSpec(t, spec2, cmd2)

	spec3 := `
		literal:show[literal:version!Hfunc()]
	`

	cmd3, err := cmd1.Merge(cmd2)
	if err != nil {
		t.Fatal(err)
	}
	AssertMatchesCommandSpec(t, spec3, cmd3)
}

func TestMergeAutocomplete(t *testing.T) {
	spec1 := `
		literal:show[param:ipv4]
	`

	cmd1, err := parseCommand("show A.B.C.D")
	if err != nil {
		t.Fatal(err)
	}

	AssertMatchesCommandSpec(t, spec1, cmd1)

	spec2 := `
		literal:show[param:ipv4!A]
	`

	cmd2, err := parseCommand("show A.B.C.D")
	if err != nil {
		t.Fatal(err)
	}

	cmd2.children[0].autocompleteFunc = func(string) []string { return nil }

	AssertMatchesCommandSpec(t, spec2, cmd2)

	spec3 := `
		literal:show[param:ipv4!A]
	`

	cmd3, err := cmd1.Merge(cmd2)
	if err != nil {
		t.Fatal(err)
	}
	AssertMatchesCommandSpec(t, spec3, cmd3)
}

func TestMergeDifferentLiterals(t *testing.T) {
	cmd1, err := parseCommand("show")
	if err != nil {
		t.Fatal(err)
	}

	cmd2, err := parseCommand("hide")
	if err != nil {
		t.Fatal(err)
	}

	cmd3, err := cmd1.Merge(cmd2)
	if err != nil {
		t.Fatal(err)
	}

	AssertMatchesCommandSpec(t, `choice[literal:show,literal:hide]`, cmd3)
}

func TestMergeDifferentAllAtoms(t *testing.T) {
	cmd1, err := parseCommand("show A.B.C.D")
	if err != nil {
		t.Fatal(err)
	}

	cmd2, err := parseCommand("show X:X:X::X")
	if err != nil {
		t.Fatal(err)
	}

	cmd3, err := parseCommand("show IFACE")
	if err != nil {
		t.Fatal(err)
	}

	cmd4, err := parseCommand("show all")
	if err != nil {
		t.Fatal(err)
	}

	cmd5, err := cmd1.Merge(cmd2, cmd3, cmd4)
	if err != nil {
		t.Fatal(err)
	}

	spec := `
		literal:show[
			choice[
				param:ipv4,
				param:ipv6,
				param:string,
				literal:all
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd5)
}

func TestMergeExplicitChoiceAndLiteral(t *testing.T) {
	cmd1, err := parseCommand("show <A.B.C.D|X:X:X::X>")
	if err != nil {
		t.Fatal(err)
	}

	spec := `
		literal:show[
			choice[
				param:ipv4,
				param:ipv6
			]
		]
	`
	AssertMatchesCommandSpec(t, spec, cmd1)

	cmd2, err := parseCommand("show version")
	if err != nil {
		t.Fatal(err)
	}

	spec = `
		literal:show[literal:version]
	`

	AssertMatchesCommandSpec(t, spec, cmd2)

	_, err = cmd1.Merge(cmd2)
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "cannot merge explicit choice \"<A.B.C.D|X:X:X::X>\" with \"version\"") {
		t.Fatalf("expected error to contain 'cannot merge explicit choice \"<A.B.C.D|X:X:X::X>\" with \"version\"', got '%s'", err.Error())
	}
}

func TestMergeExplicitChoiceAndChoice(t *testing.T) {
	cmd1, err := parseCommand("show <A.B.C.D|X:X:X::X>")
	if err != nil {
		t.Fatal(err)
	}

	spec := `
		literal:show[
			choice[
				param:ipv4,
				param:ipv6
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd1)

	cmd2, err := parseCommand("show <IFACE|all>")
	if err != nil {
		t.Fatal(err)
	}

	spec = `
		literal:show[
			choice[
				param:string,
				literal:all
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd2)

	_, err = cmd1.Merge(cmd2)
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "cannot merge explicit choice \"<A.B.C.D|X:X:X::X>\" with \"<IFACE|all>\"") {
		t.Fatalf("expected error to contain 'cannot merge explicit choice \"<A.B.C.D|X:X:X::X>\" with \"<IFACE|all>\", got '%s'", err.Error())
	}
}

func TestMergePiecemeal(t *testing.T) {
	cmd1, err := parseCommand("show A.B.C.D detail")
	if err != nil {
		t.Fatal(err)
	}

	spec := `
		literal:show[
			param:ipv4[
				literal:detail
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd1)

	cmd2, err := parseCommand("show X:X:X::X summary")
	if err != nil {
		t.Fatal(err)
	}

	spec = `
		literal:show[
			param:ipv6[
				literal:summary
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd2)

	cmd3, err := cmd1.Merge(cmd2)
	if err != nil {
		t.Fatal(err)
	}

	spec = `
		literal:show[
			choice[
				param:ipv4[
					literal:detail
				],
				param:ipv6[
					literal:summary
				]
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd3)

	cmd4, err := parseCommand("show A.B.C.D summary")
	if err != nil {
		t.Fatal(err)
	}

	spec = `
		literal:show[
			param:ipv4[
				literal:summary
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd4)

	cmd5, err := cmd3.Merge(cmd4)
	if err != nil {
		t.Fatal(err)
	}

	// Note: the two literal:summary nodes are not merged because of limitations to
	// the merging algorithm. I think this is probably not a big deal.

	spec = `
		literal:show[
			choice[
				param:ipv4[
					choice[
						literal:detail,
						literal:summary
					]
				],
				param:ipv6[
					literal:summary
				]
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd5)

	cmd6, err := parseCommand("show X:X:X::X detail")
	if err != nil {
		t.Fatal(err)
	}

	spec = `
		literal:show[
			param:ipv6[
				literal:detail
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd6)

	cmd7, err := cmd5.Merge(cmd6)
	if err != nil {
		t.Fatal(err)
	}

	spec = `
		literal:show[
			choice[
				param:ipv4[
					choice.1[
						literal:summary
						literal:detail,
					]
				],
				param:ipv6[
					choice.1
				]
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd7)
}

func TestMergePrefix(t *testing.T) {
	cmd1, err := parseCommand("show ip route")
	if err != nil {
		t.Fatal(err)
	}

	spec := `
		literal:show[
			literal:ip[
				literal:route
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd1)

	cmd2, err := parseCommand("show ip")
	if err != nil {
		t.Fatal(err)
	}

	cmd2.children[0].description = "Show IP information"

	spec = `
		literal:show[
			literal:ip?"Show IP information"
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd2)

	cmd3, err := cmd1.Merge(cmd2)
	if err != nil {
		t.Fatal(err)
	}

	spec = `
		literal:show[
			literal:ip?"Show IP information"[
				literal:route
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd3)
}

func TestMergeSuffix(t *testing.T) {
	// merge "show ip route" into "show ip", making sure to set a description on "show ip" in the second command
	cmd1, err := parseCommand("show ip route")
	if err != nil {
		t.Fatal(err)
	}

	spec := `
		literal:show[
			literal:ip[
				literal:route
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd1)

	cmd2, err := parseCommand("show ip")
	if err != nil {
		t.Fatal(err)
	}

	cmd2.children[0].description = "Show IP information"

	spec = `
		literal:show[
			literal:ip?"Show IP information"
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd2)

	cmd3, err := cmd2.Merge(cmd1)
	if err != nil {
		t.Fatal(err)
	}

	spec = `
		literal:show[
			literal:ip?"Show IP information"[
				literal:route
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd3)
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

	AssertMatchesMatchSpec(t, "show", matches)
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

	AssertMatchesMatchSpec(t, "show", matches)
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

	AssertMatchesMatchSpec(t, "string:foobar", matches)
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

	AssertMatchesMatchSpec(t, "ipv4:192.168.0.1", matches)
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

	AssertMatchesMatchSpec(t, "ipv6:2001:db8::68", matches)
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

	AssertMatchesMatchSpec(t, "ipv6:::ffff:192.168.0.1", matches)
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
	AssertMatchesMatchSpec(t, "foo", matches)

	matches = cmd.Match("bar")
	AssertMatchesMatchSpec(t, "bar", matches)

	matches = cmd.Match("baz")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchMultiple(t *testing.T) {
	cmd, err := parseCommand("foo bar baz")
	if err != nil {
		t.Fatal(err)
	}

	cmd.children[0].children[0].handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("foo bar baz")
	AssertMatchesMatchSpec(t, "foo bar baz", matches)
}

func TestMatchMultipleWithChoice(t *testing.T) {
	cmd, err := parseCommand("foo <bar|baz> qux")
	if err != nil {
		t.Fatal(err)
	}

	cmd.children[0].children[0].children[0].handlerFunc = reflect.ValueOf(func() {})
	cmd.children[0].children[1].children[0].handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("foo bar qux")
	AssertMatchesMatchSpec(t, "foo bar qux", matches)

	matches = cmd.Match("foo baz qux")
	AssertMatchesMatchSpec(t, "foo baz qux", matches)
}

func TestMatchMultipleWithString(t *testing.T) {
	cmd, err := parseCommand("before WORD after")
	if err != nil {
		t.Fatal(err)
	}

	cmd.children[0].children[0].handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("before foo after")
	AssertMatchesMatchSpec(t, "before string:foo after", matches)

	matches = cmd.Match("before bar after")
	AssertMatchesMatchSpec(t, "before string:bar after", matches)

	matches = cmd.Match("before foo")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}

	matches = cmd.Match("before bar baz after")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}

	matches = cmd.Match("after foo before")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchMultipleWithIPv4(t *testing.T) {
	cmd, err := parseCommand("show ip route A.B.C.D")
	if err != nil {
		t.Fatal(err)
	}

	cmd.children[0].children[0].children[0].handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("show ip route 1.2.3.4")
	AssertMatchesMatchSpec(t, "show ip route ipv4:1.2.3.4", matches)
}

func TestMatchChoice(t *testing.T) {
	cmd, err := parseCommand("<foo|bar>")
	if err != nil {
		t.Fatal(err)
	}

	cmd.children[0].handlerFunc = reflect.ValueOf(func() {})
	cmd.children[1].handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("foo")
	AssertMatchesMatchSpec(t, "foo", matches)

	matches = cmd.Match("bar")
	AssertMatchesMatchSpec(t, "bar", matches)

	matches = cmd.Match("baz")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchAmbiguousMatch(t *testing.T) {
	cmd, err := parseCommand("show <ip|interface>")
	if err != nil {
		t.Fatal(err)
	}

	cmd.children[0].children[0].handlerFunc = reflect.ValueOf(func() {})
	cmd.children[0].children[1].handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("sh i")
	AssertMatchesMatchSpec(t, "show ip\nshow interface", matches)
}

func TestMatchDisambiguateWithLaterToken(t *testing.T) {
	cmd, err := parseCommand("show ip route <A.B.C.D|X:X:X::X>")
	if err != nil {
		t.Fatal(err)
	}

	cmd.children[0].children[0].children[0].children[0].handlerFunc = reflect.ValueOf(func() {})
	cmd.children[0].children[0].children[0].children[1].handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("sh i ro 1.2.3.4")
	AssertMatchesMatchSpec(t, "show ip route ipv4:1.2.3.4", matches)
}

func TestMatchCommonPrefixesAreAmbiguous(t *testing.T) {
	cmd, err := parseCommand("show <ip|ipv6>")
	if err != nil {
		t.Fatal(err)
	}

	cmd.children[0].children[0].handlerFunc = reflect.ValueOf(func() {})
	cmd.children[0].children[1].handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("sh i")
	AssertMatchesMatchSpec(t, "show ip\nshow ipv6", matches)
}

func TestMatchExactMatchesAreNonAmbiguous(t *testing.T) {
	cmd, err := parseCommand("show <ip|ipv6>")
	if err != nil {
		t.Fatal(err)
	}

	cmd.children[0].children[0].handlerFunc = reflect.ValueOf(func() {})
	cmd.children[0].children[1].handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("sh ip")
	AssertMatchesMatchSpec(t, "show ip", matches)
}

func TestMatchCommonPrefixesAreAmbiguousMoreComplicated(t *testing.T) {
	cmd, err := parseCommand("show <ip|ipv6> <route|routes>")
	if err != nil {
		t.Fatal(err)
	}

	cmd.children[0].children[0].children[0].children[0].handlerFunc = reflect.ValueOf(func() {})
	cmd.children[0].children[0].children[0].children[1].handlerFunc = reflect.ValueOf(func() {})

	matches := cmd.Match("sh i r")
	spec := `
		show ip route
		show ip routes
		show ipv6 route
		show ipv6 routes
	`
	AssertMatchesMatchSpec(t, spec, matches)

	matches = cmd.Match("sh ip r")
	spec = `
		show ip route
		show ip routes
	`
	AssertMatchesMatchSpec(t, spec, matches)

	matches = cmd.Match("sh ipv r")
	spec = `
		show ipv6 route
		show ipv6 routes
	`
	AssertMatchesMatchSpec(t, spec, matches)

	matches = cmd.Match("sh i rout")
	spec = `
		show ip route
		show ip routes
		show ipv6 route
		show ipv6 routes
	`
	AssertMatchesMatchSpec(t, spec, matches)

	matches = cmd.Match("sh i route")
	spec = `
		show ip route
		show ipv6 route
	`
	AssertMatchesMatchSpec(t, spec, matches)

	matches = cmd.Match("sh i routes")
	spec = `
		show ip routes
		show ipv6 routes
	`
	AssertMatchesMatchSpec(t, spec, matches)

	matches = cmd.Match("sh ip route")
	spec = `
		show ip route
	`
	AssertMatchesMatchSpec(t, spec, matches)

	matches = cmd.Match("sh ip routes")
	spec = `
		show ip routes
	`
	AssertMatchesMatchSpec(t, spec, matches)

	matches = cmd.Match("sh ipv route")
	spec = `
		show ipv6 route
	`
	AssertMatchesMatchSpec(t, spec, matches)

	matches = cmd.Match("sh ipv routes")
	spec = `
		show ipv6 routes
	`
	AssertMatchesMatchSpec(t, spec, matches)
}

func TestHandlerNoArgs(t *testing.T) {
	cmd, err := parseCommand("show version")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()
	if len(leaves) != 1 {
		t.Fatal("expected 1 leaf")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func() error {
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerWrongArgs(t *testing.T) {
	cmd, err := parseCommand("show version")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()
	if len(leaves) != 1 {
		t.Fatal("expected 1 leaf")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(addr netip.Addr) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "expected func() error, got func(netip.Addr) error") {
		t.Fatal("error should contain 'expected func() error, got func(netip.Addr) error'")
	}
}

func TestHandlerWrongReturnValue(t *testing.T) {
	cmd, err := parseCommand("show version")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()
	if len(leaves) != 1 {
		t.Fatal("expected 1 leaf")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func() {})
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "expected func() error, got func()") {
		t.Fatal("error should contain 'expected func() error, got func()'")
	}
}

func TestHandlerArgs(t *testing.T) {
	cmd, err := parseCommand("show WORD")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()

	if len(leaves) != 1 {
		t.Fatal("expected 1 leaf")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(word string) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerWrongArgType(t *testing.T) {
	cmd, err := parseCommand("show WORD")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()

	if len(leaves) != 1 {
		t.Fatal("expected 1 leaf")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(word int) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "expected func(string) error, got func(int) error") {
		t.Fatal("error should contain 'expected func(string) error, got func(int) error'")
	}
}

func TestHandlerChoice(t *testing.T) {
	cmd, err := parseCommand("show <ip|ipv6>")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()

	if len(leaves) != 2 {
		t.Fatal("expected 2 leaves")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(ip, ipv6 bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	leaf = leaves[1]
	err = leaf.SetHandlerFunc(func(ip, ipv6 bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerChoiceWithParam(t *testing.T) {
	cmd, err := parseCommand("show <IFACE|all>")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()

	if len(leaves) != 2 {
		t.Fatal("expected 2 leaves")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(iface string, all bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	leaf = leaves[1]
	err = leaf.SetHandlerFunc(func(iface string, all bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerChoiceWithOverlappingParams(t *testing.T) {
	cmd, err := parseCommand("show <A.B.C.D|X:X:X::X|all>")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()

	if len(leaves) != 3 {
		t.Fatal("expected 3 leaves")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(addr netip.Addr, all bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	leaf = leaves[1]
	err = leaf.SetHandlerFunc(func(addr netip.Addr, all bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	leaf = leaves[2]
	err = leaf.SetHandlerFunc(func(addr netip.Addr, all bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerChoiceWithOverlappingCommandsOrderMatters(t *testing.T) {
	cmd, err := parseCommand("show <all|A.B.C.D|X:X:X::X>")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()

	if len(leaves) != 3 {
		t.Fatal("expected 3 leaves")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(all bool, addr netip.Addr) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	leaf = leaves[1]

	err = leaf.SetHandlerFunc(func(all bool, addr netip.Addr) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	leaf = leaves[2]

	err = leaf.SetHandlerFunc(func(all bool, addr netip.Addr) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerChoiceWithOverlappingCommandsOrderInterspersed(t *testing.T) {
	cmd, err := parseCommand("show <A.B.C.D|all|X:X:X::X>")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()

	if len(leaves) != 3 {
		t.Fatal("expected 3 leaves")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(addr netip.Addr, all bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	leaf = leaves[1]

	err = leaf.SetHandlerFunc(func(addr netip.Addr, all bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	leaf = leaves[2]

	err = leaf.SetHandlerFunc(func(addr netip.Addr, all bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerChoiceWithLiteralAfter(t *testing.T) {
	cmd, err := parseCommand("show <A.B.C.D|all> detail")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()

	if len(leaves) != 1 {
		t.Fatal("expected 1 leaves")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(addr netip.Addr, all bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerChoiceWithParameterAfter(t *testing.T) {
	cmd, err := parseCommand("show <A.B.C.D|all> IFACE")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()

	if len(leaves) != 1 {
		t.Fatal("expected 1 leaves")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(addr netip.Addr, all bool, iface string) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerChoiceWithChoiceAfter(t *testing.T) {
	cmd, err := parseCommand("show <A.B.C.D|all> <detail|IFACE>")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()

	if len(leaves) != 2 {
		t.Fatal("expected 2 leaves")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(addr netip.Addr, all bool, detail bool, iface string) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	leaf = leaves[1]
	err = leaf.SetHandlerFunc(func(addr netip.Addr, all bool, detail bool, iface string) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestParamTypesAfterMergeNoParams(t *testing.T) {
	cmd1, err := parseCommand("show version")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd1.Leaves()

	if len(leaves) != 1 {
		t.Fatal("expected 1 leaves")
	}

	leaf := leaves[0]

	if len(leaf.paramTypes) != 0 {
		t.Fatal("expected 0 params")
	}

	cmd2, err := parseCommand("show path")
	if err != nil {
		t.Fatal(err)
	}

	leaves = cmd2.Leaves()

	if len(leaves) != 1 {
		t.Fatal("expected 1 leaves")
	}

	leaf = leaves[0]

	if len(leaf.paramTypes) != 0 {
		t.Fatal("expected 0 params")
	}

	cmd3, err := cmd1.Merge(cmd2)
	if err != nil {
		t.Fatal(err)
	}

	leaves = cmd3.Leaves()

	if len(leaves) != 2 {
		t.Fatal("expected 2 leaves")
	}

	leaf = leaves[0]

	if len(leaf.paramTypes) != 0 {
		t.Fatal("expected 0 params")
	}

	leaf = leaves[1]

	if len(leaf.paramTypes) != 0 {
		t.Fatal("expected 0 params")
	}
}

func TestParamTypesAfterMergeWithParams(t *testing.T) {
	cmd1, err := parseCommand("show version")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd1.Leaves()

	if len(leaves) != 1 {
		t.Fatal("expected 1 leaves")
	}

	leaf := leaves[0]

	if len(leaf.paramTypes) != 0 {
		t.Fatal("expected 0 params")
	}

	cmd2, err := parseCommand("show A.B.C.D")
	if err != nil {
		t.Fatal(err)
	}

	leaves = cmd2.Leaves()

	if len(leaves) != 1 {
		t.Fatal("expected 1 leaves")
	}

	leaf = leaves[0]

	if len(leaf.paramTypes) != 1 {
		t.Fatal("expected 1 params")
	}

	cmd3, err := cmd1.Merge(cmd2)
	if err != nil {
		t.Fatal(err)
	}

	leaves = cmd3.Leaves()

	if len(leaves) != 2 {
		t.Fatal("expected 2 leaves")
	}

	leaf = leaves[0]

	if len(leaf.paramTypes) != 0 {
		t.Fatal("expected 0 params")
	}

	leaf = leaves[1]

	if len(leaf.paramTypes) != 1 {
		t.Fatal("expected 1 params")
	}
}

func TestParamTypesAfterMergeWithParamsAndChoices(t *testing.T) {
	cmd1, err := parseCommand("show bgp <A.B.C.D>")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd1.Leaves()

	if len(leaves) != 1 {
		t.Fatal("expected 1 leaves")
	}

	leaf := leaves[0]

	if !reflect.DeepEqual(leaf.paramTypes, []reflect.Type{reflect.TypeOf(netip.Addr{})}) {
		t.Fatalf("expected %v, got %v", []reflect.Type{reflect.TypeOf(netip.Addr{})}, leaf.paramTypes)
	}

	cmd2, err := parseCommand("show interface IFACE")
	if err != nil {
		t.Fatal(err)
	}

	leaves = cmd2.Leaves()

	if len(leaves) != 1 {
		t.Fatal("expected 1 leaves")
	}

	leaf = leaves[0]

	if !reflect.DeepEqual(leaf.paramTypes, []reflect.Type{reflect.TypeOf("")}) {
		t.Fatalf("expected %v, got %v", []reflect.Type{reflect.TypeOf("")}, leaf.paramTypes)
	}
}
