package commands

import (
	"io"
	"net/netip"
	"reflect"
	"strings"
	"testing"
)

func TestIsPrefixOfIPv4Address(t *testing.T) {
	if !isPrefixOfIPv4Address("") {
		t.Fatal("empty string should be a prefix of any IPv4 address")
	}

	if !isPrefixOfIPv4Address("1") {
		t.Fatal("1 should be a prefix of any IPv4 address")
	}

	if !isPrefixOfIPv4Address("9") {
		t.Fatal("9 should be a prefix of any IPv4 address")
	}

	if !isPrefixOfIPv4Address("10") {
		t.Fatal("10 should be a prefix of any IPv4 address")
	}

	if !isPrefixOfIPv4Address("99") {
		t.Fatal("99 should be a prefix of any IPv4 address")
	}

	if !isPrefixOfIPv4Address("100") {
		t.Fatal("100 should be a prefix of any IPv4 address")
	}

	if !isPrefixOfIPv4Address("255") {
		t.Fatal("255 should be a prefix of any IPv4 address")
	}

	if isPrefixOfIPv4Address("256") {
		t.Fatal("256 should not be a prefix of any IPv4 address")
	}

	if !isPrefixOfIPv4Address("1.") {
		t.Fatal("1. should be a prefix of any IPv4 address")
	}

	if !isPrefixOfIPv4Address("1.2") {
		t.Fatal("1.2 should be a prefix of any IPv4 address")
	}

	if !isPrefixOfIPv4Address("1.2.3") {
		t.Fatal("1.2.3 should be a prefix of any IPv4 address")
	}

	if !isPrefixOfIPv4Address("1.2.3.4") {
		t.Fatal("1.2.3.4 should be a prefix of any IPv4 address")
	}

	if isPrefixOfIPv4Address("1..3.4") {
		t.Fatal("1..3.4 should not be a prefix of any IPv4 address")
	}

	if isPrefixOfIPv4Address(".2.3.4") {
		t.Fatal(".2.3.4 should not be a prefix of any IPv4 address")
	}

	if isPrefixOfIPv4Address("1.2.3.4.") {
		t.Fatal("1.2.3.4. should not be a prefix of any IPv4 address")
	}

	if isPrefixOfIPv4Address("256.") {
		t.Fatal("256. should not be a prefix of any IPv4 address")
	}

	if isPrefixOfIPv4Address("256.2") {
		t.Fatal("256.2 should not be a prefix of any IPv4 address")
	}

	if isPrefixOfIPv4Address("256.2.") {
		t.Fatal("256.2. should not be a prefix of any IPv4 address")
	}

	if isPrefixOfIPv4Address("256.2.3") {
		t.Fatal("256.2.3 should not be a prefix of any IPv4 address")
	}

	if isPrefixOfIPv4Address("256.2.3.") {
		t.Fatal("256.2.3. should not be a prefix of any IPv4 address")
	}

	if isPrefixOfIPv4Address("256.2.3.4") {
		t.Fatal("256.2.3.4 should not be a prefix of any IPv4 address")
	}

	if isPrefixOfIPv4Address("1.256") {
		t.Fatal("1.256 should not be a prefix of any IPv4 address")
	}

	if isPrefixOfIPv4Address("1.256.") {
		t.Fatal("1.256. should not be a prefix of any IPv4 address")
	}

	if isPrefixOfIPv4Address("1.256.3") {
		t.Fatal("1.256.3 should not be a prefix of any IPv4 address")
	}

	if isPrefixOfIPv4Address("1.256.3.") {
		t.Fatal("1.256.3. should not be a prefix of any IPv4 address")
	}

	if isPrefixOfIPv4Address("1.256.3.4") {
		t.Fatal("1.256.3.4 should not be a prefix of any IPv4 address")
	}

	if isPrefixOfIPv4Address("1.2.256") {
		t.Fatal("1.2.256 should not be a prefix of any IPv4 address")
	}

	if isPrefixOfIPv4Address("1.2.256.") {
		t.Fatal("1.2.256. should not be a prefix of any IPv4 address")
	}

	if isPrefixOfIPv4Address("1.2.256.4") {
		t.Fatal("1.2.256.4 should not be a prefix of any IPv4 address")
	}

	if isPrefixOfIPv4Address("1.2.3.256") {
		t.Fatal("1.2.3.256 should not be a prefix of any IPv4 address")
	}
}

func TestIsPrefixOfIPv6Address(t *testing.T) {
	if !isPrefixOfIPv6Address("") {
		t.Fatal("empty string should be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("1") {
		t.Fatal("1 should be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("9") {
		t.Fatal("9 should be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("a") {
		t.Fatal("a should be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("A") {
		t.Fatal("A should be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("f") {
		t.Fatal("f should be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("F") {
		t.Fatal("F should be a prefix of an IPv6 address")
	}

	if isPrefixOfIPv6Address("g") {
		t.Fatal("g should not be a prefix of an IPv6 address")
	}

	if isPrefixOfIPv6Address("G") {
		t.Fatal("G should not be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address(":") {
		t.Fatal(": should be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("::") {
		t.Fatal(":: should be a prefix of an IPv6 address")
	}

	if isPrefixOfIPv6Address(":::") {
		t.Fatal("::: should not be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("::1") {
		t.Fatal("::1 should be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("::ffff") {
		t.Fatal("::ffff should be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("::ffff:") {
		t.Fatal("::ffff: should be a prefix of an IPv6 address")
	}

	if isPrefixOfIPv6Address("::ffff::") {
		t.Fatal("::ffff:: should not be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("::1:") {
		t.Fatal("::1: should be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("ffff") {
		t.Fatal("ffff should be a prefix of an IPv6 address")
	}

	if isPrefixOfIPv6Address("10000") {
		t.Fatal("10000 should not be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("1:") {
		t.Fatal("1: should be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("1::") {
		t.Fatal("1:: should be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("ffff:") {
		t.Fatal("ffff: should be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("ffff::") {
		t.Fatal("ffff:: should be a prefix of an IPv6 address")
	}

	if isPrefixOfIPv6Address("10000:") {
		t.Fatal("10000: should not be a prefix of an IPv6 address")
	}

	if isPrefixOfIPv6Address("10000::") {
		t.Fatal("10000:: should not be a prefix of an IPv6 address")
	}

	if isPrefixOfIPv6Address("1:2:3:4:5:6:7:8:9") {
		t.Fatal("1:2:3:4:5:6:7:8:9 should not be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("1:2:3:4:5:6:7::") {
		t.Fatal("1:2:3:4:5:6:7:: should be a prefix of an IPv6 address")
	}

	if isPrefixOfIPv6Address("1:2:3:4:5:6:7::8") {
		t.Fatal("1:2:3:4:5:6:7::8 should not be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("::ffff:1.") {
		t.Fatal("::ffff:1. should be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("::ffff:1.2.") {
		t.Fatal("::ffff:1.2. should be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("::ffff:1.2.3.") {
		t.Fatal("::ffff:1.2.3. should be a prefix of an IPv6 address")
	}

	if !isPrefixOfIPv6Address("::ffff:1.2.3.4") {
		t.Fatal("::ffff:1.2.3.4 should be a prefix of an IPv6 address")
	}

	if isPrefixOfIPv6Address("::ffff:256.") {
		t.Fatal("::ffff:256. should not be a prefix of an IPv6 address")
	}
}

func TestParseCommand(t *testing.T) {
	s := "show version"
	spec := `
		literal:show[literal:version]
	`
	cmd, err := ParseDeclaration(s)
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

	cmd, err := ParseDeclaration(s)
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

	cmd, err := ParseDeclaration(s)
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

	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

	AssertMatchesCommandSpec(t, spec, cmd)
}

func TestMergeDescription(t *testing.T) {
	spec1 := `
		literal:show[literal:version]
	`

	cmd1, err := ParseDeclaration("show version")
	if err != nil {
		t.Fatal(err)
	}

	AssertMatchesCommandSpec(t, spec1, cmd1)

	spec2 := `
		literal:show[literal:version?"Show version information"]
	`

	cmd2, err := ParseDeclaration("show version")
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

	cmd1, err := ParseDeclaration("show version")
	if err != nil {
		t.Fatal(err)
	}

	AssertMatchesCommandSpec(t, spec1, cmd1)

	spec2 := `
		literal:show[literal:version!Hfunc()]
	`

	cmd2, err := ParseDeclaration("show version")
	if err != nil {
		t.Fatal(err)
	}

	err = cmd2.children[0].SetHandlerFunc(func(w io.Writer) error { return nil })
	if err != nil {
		t.Fatal(err)
	}

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

	cmd1, err := ParseDeclaration("show A.B.C.D")
	if err != nil {
		t.Fatal(err)
	}

	AssertMatchesCommandSpec(t, spec1, cmd1)

	spec2 := `
		literal:show[param:ipv4!A]
	`

	cmd2, err := ParseDeclaration("show A.B.C.D")
	if err != nil {
		t.Fatal(err)
	}

	cmd2.children[0].autocompleteFunc = func() ([]string, error) { return nil, nil }

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
	cmd1, err := ParseDeclaration("show")
	if err != nil {
		t.Fatal(err)
	}

	cmd2, err := ParseDeclaration("hide")
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
	cmd1, err := ParseDeclaration("show A.B.C.D")
	if err != nil {
		t.Fatal(err)
	}

	cmd2, err := ParseDeclaration("show X:X:X::X")
	if err != nil {
		t.Fatal(err)
	}

	cmd3, err := ParseDeclaration("show IFACE")
	if err != nil {
		t.Fatal(err)
	}

	cmd4, err := ParseDeclaration("show all")
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
	cmd1, err := ParseDeclaration("show <A.B.C.D|X:X:X::X>")
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

	cmd2, err := ParseDeclaration("show version")
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
	cmd1, err := ParseDeclaration("show <A.B.C.D|X:X:X::X>")
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

	cmd2, err := ParseDeclaration("show <IFACE|all>")
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

func TestMergeExplicitChoiceIsAllowedIfChildrenAreTheSame(t *testing.T) {
	cmd1, err := ParseDeclaration("show <A.B.C.D|X:X:X::X>")
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

	cmd2, err := ParseDeclaration("show <A.B.C.D|X:X:X::X>")
	if err != nil {
		t.Fatal(err)
	}

	spec = `
		literal:show[
			choice[
				param:ipv4,
				param:ipv6
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd2)

	cmd3, err := cmd1.Merge(cmd2)
	if err != nil {
		t.Fatal(err)
	}

	AssertMatchesCommandSpec(t, spec, cmd3)
}

func TestMergeExplicitChoiceSameChildrenWithAttributes(t *testing.T) {
	cmd1, err := ParseDeclaration("show <A.B.C.D|X:X:X::X>")
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

	cmd2, err := ParseDeclaration("show <A.B.C.D|X:X:X::X>")
	if err != nil {
		t.Fatal(err)
	}

	cmd2.children[0].children[0].description = "Show IP address"
	cmd2.children[0].children[1].description = "Show IPv6 address"

	spec = `
		literal:show[
			choice[
				param:ipv4?"Show IP address",
				param:ipv6?"Show IPv6 address"
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd2)

	cmd3, err := cmd1.Merge(cmd2)
	if err != nil {
		t.Fatal(err)
	}

	AssertMatchesCommandSpec(t, spec, cmd3)
}

func TestMergeExplicitChoiceSameChildrenWithDescendents(t *testing.T) {
	cmd1, err := ParseDeclaration("show <A.B.C.D|X:X:X::X> detail")
	if err != nil {
		t.Fatal(err)
	}

	spec := `
		literal:show[
			choice[
				param:ipv4[
					literal:detail.1
				],
				param:ipv6[
					literal:detail.1
				]
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd1)

	cmd2, err := ParseDeclaration("show <A.B.C.D|X:X:X::X> summary")
	if err != nil {
		t.Fatal(err)
	}

	spec = `
		literal:show[
			choice[
				param:ipv4[
					literal:summary.1
				],
				param:ipv6[
					literal:summary.1
				]
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
					choice.1[
						literal:detail.1,
						literal:summary.1
					],
				],
				param:ipv6[
					choice.1
				]
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd3)
}

func TestMergeSeparatePaths(t *testing.T) {
	cmd1, err := ParseDeclaration("show ip route A.B.C.D")
	if err != nil {
		t.Fatal(err)
	}

	spec := `
		literal:show[
			literal:ip[
				literal:route[
					param:ipv4
				]
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd1)

	cmd2, err := ParseDeclaration("show ipv6 route X:X:X::X")
	if err != nil {
		t.Fatal(err)
	}

	spec = `
		literal:show[
			literal:ipv6[
				literal:route[
					param:ipv6
				]
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
				literal:ip[
					literal:route[
						param:ipv4
					]
				],
				literal:ipv6[
					literal:route[
						param:ipv6
					]
				]
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd3)
}

func TestMergePiecemeal(t *testing.T) {
	cmd1, err := ParseDeclaration("show A.B.C.D detail")
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

	cmd2, err := ParseDeclaration("show X:X:X::X summary")
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

	cmd4, err := ParseDeclaration("show A.B.C.D summary")
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

	cmd6, err := ParseDeclaration("show X:X:X::X detail")
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
					choice[
						literal:detail,
						literal:summary
					]
				],
				param:ipv6[
					choice[
						literal:summary,
						literal:detail
					]
				]
			]
		]
	`

	AssertMatchesCommandSpec(t, spec, cmd7)
}

func TestMergePrefix(t *testing.T) {
	cmd1, err := ParseDeclaration("show ip route")
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

	cmd2, err := ParseDeclaration("show ip")
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
	cmd1, err := ParseDeclaration("show ip route")
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

	cmd2, err := ParseDeclaration("show ip")
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
	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("show")
	if matches == nil {
		t.Fatal("expected match")
	}

	AssertMatchesMatchSpec(t, "show", matches)

	if len(matches[0].args) != 0 {
		t.Fatalf("expected no args, got %v", matches[0].args)
	}
}

func TestMatchLiteralNoHandler(t *testing.T) {
	s := "show"
	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("show")

	AssertMatchesMatchSpec(t, "show", matches)

	if matches[0].isComplete {
		t.Fatal("expected incomplete match")
	}
}

func TestMatchLiteralPrefix(t *testing.T) {
	s := "show"
	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("sh")
	if len(matches) != 1 {
		t.Fatal("expected 1 match")
	}

	AssertMatchesMatchSpec(t, "show", matches)
}

func TestMatchLiteralInvalid(t *testing.T) {
	s := "show"
	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("sha")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchString(t *testing.T) {
	s := "WORD"
	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("foobar")
	if len(matches) != 1 {
		t.Fatal("expected match")
	}

	AssertMatchesMatchSpec(t, "string:foobar", matches)
}

func TestMatchStringNoHandler(t *testing.T) {
	s := "WORD"
	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("foobar")

	AssertMatchesMatchSpec(t, "string:foobar", matches)

	if matches[0].isComplete {
		t.Fatal("expected incomplete match")
	}
}

func TestMatchIPv4(t *testing.T) {
	s := "A.B.C.D"
	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("192.168.0.1")
	if len(matches) != 1 {
		t.Fatal("expected match")
	}

	AssertMatchesMatchSpec(t, "ipv4:192.168.0.1", matches)
}

func TestMatchIPv4NoHandler(t *testing.T) {
	s := "A.B.C.D"
	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("192.168.0.1")

	AssertMatchesMatchSpec(t, "ipv4:192.168.0.1", matches)

	if matches[0].isComplete {
		t.Fatal("expected incomplete match")
	}
}

func TestMatchIPv4IncompleteAddr(t *testing.T) {
	s := "A.B.C.D"
	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("192.168.0")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchIPv4InvalidAddr(t *testing.T) {
	s := "A.B.C.D"
	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("300.0.0.1")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchIPv4Nonsense(t *testing.T) {
	s := "A.B.C.D"
	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("foobar")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchIPv4NoMatchIPv6(t *testing.T) {
	s := "A.B.C.D"
	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("2001:db8::68")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchIPv6(t *testing.T) {
	s := "X:X:X::X"
	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("2001:db8::68")
	if len(matches) != 1 {
		t.Fatal("expected match")
	}

	AssertMatchesMatchSpec(t, "ipv6:2001:db8::68", matches)
}

func TestMatchIPv6NoHandler(t *testing.T) {
	s := "X:X:X::X"
	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("2001:db8::68")

	AssertMatchesMatchSpec(t, "ipv6:2001:db8::68", matches)

	if matches[0].isComplete {
		t.Fatal("expected incomplete match")
	}
}

func TestMatchIPv6IncompleteAddr(t *testing.T) {
	s := "X:X:X::X"
	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("2001:db8")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchIPv6InvalidAddr(t *testing.T) {
	s := "X:X:X::X"
	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("2001:db8::g")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchIPv6Nonsense(t *testing.T) {
	s := "X:X:X::X"
	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("foobar")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchIPv6NoMatchIPv4(t *testing.T) {
	s := "X:X:X::X"
	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("192.168.0.1")
	if len(matches) != 0 {
		t.Fatal("expected no match")
	}
}

func TestMatchIPv6MappedIPv4(t *testing.T) {
	s := "X:X:X::X"
	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("::ffff:192.168.0.1")
	if len(matches) != 1 {
		t.Fatal("expected match")
	}

	AssertMatchesMatchSpec(t, "ipv6:::ffff:192.168.0.1", matches)
}

func TestMatchChoiceLiteral(t *testing.T) {
	s := "<foo|bar>"
	cmd, err := ParseDeclaration(s)
	if err != nil {
		t.Fatal(err)
	}

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
	cmd, err := ParseDeclaration("foo bar baz")
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("foo bar baz")
	AssertMatchesMatchSpec(t, "foo bar baz", matches)
}

func TestMatchMultipleWithChoice(t *testing.T) {
	cmd, err := ParseDeclaration("foo <bar|baz> qux")
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("foo bar qux")
	AssertMatchesMatchSpec(t, "foo bar qux", matches)

	matches = cmd.Match("foo baz qux")
	AssertMatchesMatchSpec(t, "foo baz qux", matches)
}

func TestMatchMultipleWithString(t *testing.T) {
	cmd, err := ParseDeclaration("before WORD after")
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("before foo after")
	AssertMatchesMatchSpec(t, "before string:foo after", matches)

	matches = cmd.Match("before bar after")
	AssertMatchesMatchSpec(t, "before string:bar after", matches)

	matches = cmd.Match("before foo")
	AssertMatchesMatchSpec(t, "before string:foo", matches)

	if matches[0].isComplete {
		t.Fatal("expected incomplete match")
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
	cmd, err := ParseDeclaration("show ip route A.B.C.D")
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("show ip route 1.2.3.4")
	AssertMatchesMatchSpec(t, "show ip route ipv4:1.2.3.4", matches)
}

func TestMatchChoice(t *testing.T) {
	cmd, err := ParseDeclaration("<foo|bar>")
	if err != nil {
		t.Fatal(err)
	}

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
	cmd, err := ParseDeclaration("show <ip|interface>")
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("sh i")
	AssertMatchesMatchSpec(t, "show ip\nshow interface", matches)
}

func TestMatchDisambiguateWithLaterToken(t *testing.T) {
	cmd, err := ParseDeclaration("show ip route <A.B.C.D|X:X:X::X>")
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("sh i ro 1.2.3.4")
	AssertMatchesMatchSpec(t, "show ip route ipv4:1.2.3.4", matches)
}

func TestMatchCommonPrefixesAreAmbiguous(t *testing.T) {
	cmd, err := ParseDeclaration("show <ip|ipv6>")
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("sh i")
	AssertMatchesMatchSpec(t, "show ip\nshow ipv6", matches)
}

func TestMatchExactMatchesAreNonAmbiguous(t *testing.T) {
	cmd, err := ParseDeclaration("show <ip|ipv6>")
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd.Match("sh ip")
	AssertMatchesMatchSpec(t, "show ip", matches)
}

func TestMatchCommonPrefixesAreAmbiguousMoreComplicated(t *testing.T) {
	cmd, err := ParseDeclaration("show <ip|ipv6> <route|routes>")
	if err != nil {
		t.Fatal(err)
	}

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

func TestFutureTokensResolveAmbiguity(t *testing.T) {
	cmd1, err := ParseDeclaration("show ip route")
	if err != nil {
		t.Fatal(err)
	}

	cmd2, err := ParseDeclaration("show ipv6 runner")
	if err != nil {
		t.Fatal(err)
	}

	cmd3, err := cmd1.Merge(cmd2)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd3.Match("sh i r")

	spec := `
		show ip route
		show ipv6 runner
	`

	AssertMatchesMatchSpec(t, spec, matches)

	matches = cmd3.Match("sh i ro")

	spec = `
		show ip route
	`

	AssertMatchesMatchSpec(t, spec, matches)

	matches = cmd3.Match("sh i ru")

	spec = `
		show ipv6 runner
	`

	AssertMatchesMatchSpec(t, spec, matches)
}

func TestFutureTokensResolveAmbiguityWithParams(t *testing.T) {
	cmd1, err := ParseDeclaration("show ip route A.B.C.D")
	if err != nil {
		t.Fatal(err)
	}

	cmd2, err := ParseDeclaration("show ipv6 route X:X:X::X")
	if err != nil {
		t.Fatal(err)
	}

	cmd3, err := cmd1.Merge(cmd2)
	if err != nil {
		t.Fatal(err)
	}

	matches := cmd3.Match("sh i r 1.1.1.1")

	spec := `
		show ip route ipv4:1.1.1.1
	`

	AssertMatchesMatchSpec(t, spec, matches)

	matches = cmd3.Match("sh i r 2001:0db8::1")

	spec = `
		show ipv6 route ipv6:2001:0db8::1
	`

	AssertMatchesMatchSpec(t, spec, matches)
}

func TestHandlerNoArgs(t *testing.T) {
	cmd, err := ParseDeclaration("show version")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()
	if len(leaves) != 1 {
		t.Fatal("expected 1 leaf")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(io.Writer) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerWrongArgs(t *testing.T) {
	cmd, err := ParseDeclaration("show version")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()
	if len(leaves) != 1 {
		t.Fatal("expected 1 leaf")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(w io.Writer, addr netip.Addr) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "expected func(io.Writer) error, got func(io.Writer, netip.Addr) error") {
		t.Fatal("error should contain 'expected func(io.Writer) error, got func(io.Writer, netip.Addr) error'")
	}
}

func TestHandlerWrongReturnValue(t *testing.T) {
	cmd, err := ParseDeclaration("show version")
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

	if !strings.Contains(err.Error(), "expected func(io.Writer) error, got func()") {
		t.Fatalf("error should contain 'expected func(io.Writer) error, got func()', got %q", err.Error())
	}
}

func TestHandlerArgs(t *testing.T) {
	cmd, err := ParseDeclaration("show WORD")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()

	if len(leaves) != 1 {
		t.Fatal("expected 1 leaf")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(w io.Writer, word string) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerWrongArgType(t *testing.T) {
	cmd, err := ParseDeclaration("show WORD")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()

	if len(leaves) != 1 {
		t.Fatal("expected 1 leaf")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(w io.Writer, word int) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "expected func(io.Writer, string) error, got func(io.Writer, int) error") {
		t.Fatalf("error should contain 'expected func(io.Writer, string) error, got func(io.Writer, int) error', got %q", err.Error())
	}
}

func TestHandlerChoice(t *testing.T) {
	cmd, err := ParseDeclaration("show <ip|ipv6>")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()

	if len(leaves) != 2 {
		t.Fatal("expected 2 leaves")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(w io.Writer, ip, ipv6 bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	leaf = leaves[1]
	err = leaf.SetHandlerFunc(func(w io.Writer, ip, ipv6 bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerChoiceWithParam(t *testing.T) {
	cmd, err := ParseDeclaration("show <IFACE|all>")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()

	if len(leaves) != 2 {
		t.Fatal("expected 2 leaves")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(w io.Writer, iface string, all bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	leaf = leaves[1]
	err = leaf.SetHandlerFunc(func(w io.Writer, iface string, all bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerChoiceWithOverlappingParams(t *testing.T) {
	cmd, err := ParseDeclaration("show <A.B.C.D|X:X:X::X|all>")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()

	if len(leaves) != 3 {
		t.Fatal("expected 3 leaves")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(w io.Writer, addr netip.Addr, all bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	leaf = leaves[1]
	err = leaf.SetHandlerFunc(func(w io.Writer, addr netip.Addr, all bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	leaf = leaves[2]
	err = leaf.SetHandlerFunc(func(w io.Writer, addr netip.Addr, all bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerChoiceWithOverlappingCommandsOrderMatters(t *testing.T) {
	cmd, err := ParseDeclaration("show <all|A.B.C.D|X:X:X::X>")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()

	if len(leaves) != 3 {
		t.Fatal("expected 3 leaves")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(w io.Writer, all bool, addr netip.Addr) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	leaf = leaves[1]

	err = leaf.SetHandlerFunc(func(w io.Writer, all bool, addr netip.Addr) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	leaf = leaves[2]

	err = leaf.SetHandlerFunc(func(w io.Writer, all bool, addr netip.Addr) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerChoiceWithOverlappingCommandsOrderInterspersed(t *testing.T) {
	cmd, err := ParseDeclaration("show <A.B.C.D|all|X:X:X::X>")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()

	if len(leaves) != 3 {
		t.Fatal("expected 3 leaves")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(w io.Writer, addr netip.Addr, all bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	leaf = leaves[1]

	err = leaf.SetHandlerFunc(func(w io.Writer, addr netip.Addr, all bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	leaf = leaves[2]

	err = leaf.SetHandlerFunc(func(w io.Writer, addr netip.Addr, all bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerChoiceWithLiteralAfter(t *testing.T) {
	cmd, err := ParseDeclaration("show <A.B.C.D|all> detail")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()

	if len(leaves) != 1 {
		t.Fatal("expected 1 leaves")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(w io.Writer, addr netip.Addr, all bool) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerChoiceWithParameterAfter(t *testing.T) {
	cmd, err := ParseDeclaration("show <A.B.C.D|all> IFACE")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()

	if len(leaves) != 1 {
		t.Fatal("expected 1 leaves")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(w io.Writer, addr netip.Addr, all bool, iface string) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerChoiceWithChoiceAfter(t *testing.T) {
	cmd, err := ParseDeclaration("show <A.B.C.D|all> <detail|IFACE>")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd.Leaves()

	if len(leaves) != 2 {
		t.Fatal("expected 2 leaves")
	}

	leaf := leaves[0]

	err = leaf.SetHandlerFunc(func(w io.Writer, addr netip.Addr, all bool, detail bool, iface string) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	leaf = leaves[1]
	err = leaf.SetHandlerFunc(func(w io.Writer, addr netip.Addr, all bool, detail bool, iface string) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestParamTypesAfterMergeNoParams(t *testing.T) {
	cmd1, err := ParseDeclaration("show version")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd1.Leaves()

	if len(leaves) != 1 {
		t.Fatal("expected 1 leaves")
	}

	leaf := leaves[0]

	if !reflect.DeepEqual(leaf.paramTypes, []reflect.Type{reflect.TypeOf((*io.Writer)(nil)).Elem()}) {
		t.Fatalf("expected %v, got %v", []reflect.Type{reflect.TypeOf((*io.Writer)(nil)).Elem()}, leaf.paramTypes)
	}

	cmd2, err := ParseDeclaration("show path")
	if err != nil {
		t.Fatal(err)
	}

	leaves = cmd2.Leaves()

	if len(leaves) != 1 {
		t.Fatal("expected 1 leaves")
	}

	leaf = leaves[0]

	if !reflect.DeepEqual(leaf.paramTypes, []reflect.Type{reflect.TypeOf((*io.Writer)(nil)).Elem()}) {
		t.Fatalf("expected %v, got %v", []reflect.Type{reflect.TypeOf((*io.Writer)(nil)).Elem()}, leaf.paramTypes)
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

	if !reflect.DeepEqual(leaf.paramTypes, []reflect.Type{reflect.TypeOf((*io.Writer)(nil)).Elem()}) {
		t.Fatalf("expected %v, got %v", []reflect.Type{reflect.TypeOf((*io.Writer)(nil)).Elem()}, leaf.paramTypes)
	}

	leaf = leaves[1]

	if !reflect.DeepEqual(leaf.paramTypes, []reflect.Type{reflect.TypeOf((*io.Writer)(nil)).Elem()}) {
		t.Fatalf("expected %v, got %v", []reflect.Type{reflect.TypeOf((*io.Writer)(nil)).Elem()}, leaf.paramTypes)
	}
}

func TestParamTypesAfterMergeWithParams(t *testing.T) {
	cmd1, err := ParseDeclaration("show version")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd1.Leaves()

	if len(leaves) != 1 {
		t.Fatal("expected 1 leaves")
	}

	leaf := leaves[0]

	if len(leaf.paramTypes) != 1 {
		t.Fatal("expected 1 param")
	}

	cmd2, err := ParseDeclaration("show A.B.C.D")
	if err != nil {
		t.Fatal(err)
	}

	leaves = cmd2.Leaves()

	if len(leaves) != 1 {
		t.Fatal("expected 1 leaves")
	}

	leaf = leaves[0]

	if len(leaf.paramTypes) != 2 {
		t.Fatal("expected 2 params")
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

	if len(leaf.paramTypes) != 1 {
		t.Fatal("expected 1 params")
	}

	leaf = leaves[1]

	if len(leaf.paramTypes) != 2 {
		t.Fatal("expected 2 params")
	}
}

func TestParamTypesAfterMergeWithParamsAndChoices(t *testing.T) {
	cmd1, err := ParseDeclaration("show bgp <A.B.C.D>")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd1.Leaves()

	if len(leaves) != 1 {
		t.Fatal("expected 1 leaves")
	}

	leaf := leaves[0]

	expected := []reflect.Type{reflect.TypeOf((*io.Writer)(nil)).Elem(), reflect.TypeOf(netip.Addr{})}
	if !reflect.DeepEqual(leaf.paramTypes, expected) {
		t.Fatalf("expected %v, got %v", expected, leaf.paramTypes)
	}

	cmd2, err := ParseDeclaration("show interface IFACE")
	if err != nil {
		t.Fatal(err)
	}

	leaves = cmd2.Leaves()

	if len(leaves) != 1 {
		t.Fatal("expected 1 leaves")
	}

	leaf = leaves[0]

	expected = []reflect.Type{reflect.TypeOf((*io.Writer)(nil)).Elem(), reflect.TypeOf("")}
	if !reflect.DeepEqual(leaf.paramTypes, expected) {
		t.Fatalf("expected %v, got %v", expected, leaf.paramTypes)
	}
}

func TestAutocompleteSimple(t *testing.T) {
	cmd1, err := ParseDeclaration("show version")
	if err != nil {
		t.Fatal(err)
	}

	cmd2, err := ParseDeclaration("reset counters")
	if err != nil {
		t.Fatal(err)
	}

	cmd3, err := cmd1.Merge(cmd2)
	if err != nil {
		t.Fatal(err)
	}

	w := &strings.Builder{}
	options, offset, err := cmd3.GetAutocompleteOptions(w, "sh")
	if err != nil {
		t.Fatal(err)
	}

	if offset != 2 {
		t.Fatalf("expected offset 2, got %d", offset)
	}

	expected := []string{"show"}

	if !reflect.DeepEqual(options, expected) {
		t.Fatalf("expected %v, got %v", expected, options)
	}
}

func TestAutocompleteEmpty(t *testing.T) {
	cmd1, err := ParseDeclaration("show version")
	if err != nil {
		t.Fatal(err)
	}

	cmd2, err := ParseDeclaration("reset counters")
	if err != nil {
		t.Fatal(err)
	}

	cmd3, err := cmd1.Merge(cmd2)
	if err != nil {
		t.Fatal(err)
	}

	w := &strings.Builder{}
	options, offset, err := cmd3.GetAutocompleteOptions(w, "")
	if err != nil {
		t.Fatal(err)
	}

	if offset != 0 {
		t.Fatalf("expected offset 0, got %d", offset)
	}

	expected := []string{"show", "reset"}

	if !reflect.DeepEqual(options, expected) {
		t.Fatalf("expected %v, got %v", expected, options)
	}
}

func TestAutocompleteMultiple(t *testing.T) {
	cmd1, err := ParseDeclaration("show interface")
	if err != nil {
		t.Fatal(err)
	}

	cmd2, err := ParseDeclaration("show ip")
	if err != nil {
		t.Fatal(err)
	}

	cmd3, err := ParseDeclaration("show ipv6")
	if err != nil {
		t.Fatal(err)
	}

	cmd4, err := ParseDeclaration("show version")
	if err != nil {
		t.Fatal(err)
	}

	cmd5, err := cmd1.Merge(cmd2, cmd3, cmd4)
	if err != nil {
		t.Fatal(err)
	}

	w := &strings.Builder{}
	options, offset, err := cmd5.GetAutocompleteOptions(w, "show ")
	if err != nil {
		t.Fatal(err)
	}

	if offset != 0 {
		t.Fatalf("expected offset 0, got %d", offset)
	}

	expected := []string{"interface", "ip", "ipv6", "version"}

	if !reflect.DeepEqual(options, expected) {
		t.Fatalf("expected %v, got %v", expected, options)
	}

	w = &strings.Builder{}
	options, offset, err = cmd5.GetAutocompleteOptions(w, "show i")
	if err != nil {
		t.Fatal(err)
	}

	if offset != 1 {
		t.Fatalf("expected offset 1, got %d", offset)
	}

	expected = []string{"interface", "ip", "ipv6"}

	if !reflect.DeepEqual(options, expected) {
		t.Fatalf("expected %v, got %v", expected, options)
	}

	w = &strings.Builder{}
	options, offset, err = cmd5.GetAutocompleteOptions(w, "show ip")
	if err != nil {
		t.Fatal(err)
	}

	if offset != 2 {
		t.Fatalf("expected offset 2, got %d", offset)
	}

	expected = []string{"ip", "ipv6"}

	if !reflect.DeepEqual(options, expected) {
		t.Fatalf("expected %v, got %v", expected, options)
	}

	w = &strings.Builder{}
	options, offset, err = cmd5.GetAutocompleteOptions(w, "show ipv")
	if err != nil {
		t.Fatal(err)
	}

	if offset != 3 {
		t.Fatalf("expected offset 3, got %d", offset)
	}

	expected = []string{"ipv6"}

	if !reflect.DeepEqual(options, expected) {
		t.Fatalf("expected %v, got %v", expected, options)
	}

	w = &strings.Builder{}
	options, offset, err = cmd5.GetAutocompleteOptions(w, "show ipv6")
	if err != nil {
		t.Fatal(err)
	}

	if offset != 4 {
		t.Fatalf("expected offset 4, got %d", offset)
	}

	expected = []string{"ipv6"}

	if !reflect.DeepEqual(options, expected) {
		t.Fatalf("expected %v, got %v", expected, options)
	}

	w = &strings.Builder{}
	options, offset, err = cmd5.GetAutocompleteOptions(w, "show q")
	if err != nil {
		t.Fatal(err)
	}

	if offset != 1 {
		t.Fatalf("expected offset 1, got %d", offset)
	}

	if len(options) != 0 {
		t.Fatalf("expected no options, got %v", options)
	}
}

func TestAutocompleteMultiLevel(t *testing.T) {
	cmd1, err := ParseDeclaration("show ip route")
	if err != nil {
		t.Fatal(err)
	}

	w := &strings.Builder{}
	options, offset, err := cmd1.GetAutocompleteOptions(w, "show ip ")
	if err != nil {
		t.Fatal(err)
	}

	if offset != 0 {
		t.Fatalf("expected offset 0, got %d", offset)
	}

	expected := []string{"route"}

	if !reflect.DeepEqual(options, expected) {
		t.Fatalf("expected %v, got %v", expected, options)
	}
}

func TestAutocompleteWithParam(t *testing.T) {
	cmd1, err := ParseDeclaration("show bgp neighbors A.B.C.D")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd1.Leaves()

	if len(leaves) != 1 {
		t.Fatalf("expected 1 leaf, got %d", len(leaves))
	}

	leaf := leaves[0]

	leaf.SetAutocompleteFunc(func() ([]string, error) {
		return []string{"1.1.1.1", "8.8.8.8"}, nil
	})

	w := &strings.Builder{}
	options, offset, err := cmd1.GetAutocompleteOptions(w, "show bgp neighbors ")
	if err != nil {
		t.Fatal(err)
	}

	if offset != 0 {
		t.Fatalf("expected offset 0, got %d", offset)
	}

	expected := []string{"1.1.1.1", "8.8.8.8"}

	if !reflect.DeepEqual(options, expected) {
		t.Fatalf("expected %v, got %v", expected, options)
	}

	w = &strings.Builder{}
	options, offset, err = cmd1.GetAutocompleteOptions(w, "show bgp neighbors 1")
	if err != nil {
		t.Fatal(err)
	}

	if offset != 1 {
		t.Fatalf("expected offset 1, got %d", offset)
	}

	expected = []string{"1.1.1.1"}

	if !reflect.DeepEqual(options, expected) {
		t.Fatalf("expected %v, got %v", expected, options)
	}

	w = &strings.Builder{}
	options, offset, err = cmd1.GetAutocompleteOptions(w, "show bgp neighbors 8.8.")
	if err != nil {
		t.Fatal(err)
	}

	if offset != 4 {
		t.Fatalf("expected offset 4, got %d", offset)
	}

	expected = []string{"8.8.8.8"}

	if !reflect.DeepEqual(options, expected) {
		t.Fatalf("expected %v, got %v", expected, options)
	}

	w = &strings.Builder{}
	options, offset, err = cmd1.GetAutocompleteOptions(w, "show bgp neighbors 2")
	if err != nil {
		t.Fatal(err)
	}

	if offset != 1 {
		t.Fatalf("expected offset 1, got %d", offset)
	}

	if len(options) != 0 {
		t.Fatalf("expected no options, got %v", options)
	}
}

func TestAutocompleteWithParamAndLiteral(t *testing.T) {
	cmd1, err := ParseDeclaration("show bgp neighbors <A.B.C.D|all>")
	if err != nil {
		t.Fatal(err)
	}

	cmd2, err := ParseDeclaration("show bgp neighbors A.B.C.D")
	if err != nil {
		t.Fatal(err)
	}

	leaves := cmd2.Leaves()

	if len(leaves) != 1 {
		t.Fatalf("expected 1 leaf, got %d", len(leaves))
	}

	leaf := leaves[0]

	leaf.SetAutocompleteFunc(func() ([]string, error) {
		return []string{"1.1.1.1", "8.8.8.8"}, nil
	})

	cmd3, err := cmd1.MergeWithoutExplicitChoiceRestrictions(cmd2)
	if err != nil {
		t.Fatal(err)
	}

	w := &strings.Builder{}
	options, offset, err := cmd3.GetAutocompleteOptions(w, "show bgp neighbors ")
	if err != nil {
		t.Fatal(err)
	}

	if offset != 0 {
		t.Fatalf("expected offset 0, got %d", offset)
	}

	expected := []string{"1.1.1.1", "8.8.8.8", "all"}

	if !reflect.DeepEqual(options, expected) {
		t.Fatalf("expected %v, got %v", expected, options)
	}
}
