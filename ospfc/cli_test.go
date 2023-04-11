package main

import (
	"fmt"
	"io"
	"net/netip"
	"strings"
	"testing"
)

func TestBuiltInExitCommand(t *testing.T) {
	cli := NewCLI()

	cli.running = true

	w := &strings.Builder{}
	cli.runLine("exit", w)

	if cli.running {
		t.Fatal("CLI should not be running")
	}
}

func TestBuiltInQuitCommand(t *testing.T) {
	cli := NewCLI()

	cli.running = true

	w := &strings.Builder{}
	cli.runLine("quit", w)

	if cli.running {
		t.Fatal("CLI should not be running")
	}
}

func TestEmptyInput(t *testing.T) {
	cli := NewCLI()

	w := &strings.Builder{}
	cli.runLine("", w)

	if w.String() != "" {
		t.Fatalf("Unexpected output: %s", w.String())
	}
}

func TestEmptyInputWithWhitespace(t *testing.T) {
	cli := NewCLI()

	w := &strings.Builder{}
	cli.runLine(" ", w)

	if w.String() != "" {
		t.Fatalf("Unexpected output: %s", w.String())
	}
}

func TestSimpleCommand(t *testing.T) {
	cli := NewCLI()

	err := cli.Register("show version", "Show version information", func(w io.Writer) error {
		fmt.Fprintf(w, "Version 1.0.0\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	w := &strings.Builder{}
	cli.runLine("show version", w)

	if w.String() != "Version 1.0.0\n" {
		t.Fatalf("Unexpected output: %s", w.String())
	}
}

func TestSimpleCommandPrefixMatching(t *testing.T) {
	cli := NewCLI()

	err := cli.Register("show version", "Show version information", func(w io.Writer) error {
		fmt.Fprintf(w, "Version 1.0.0\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	w := &strings.Builder{}
	cli.runLine("sh ver", w)

	if w.String() != "Version 1.0.0\n" {
		t.Fatalf("Unexpected output: %s", w.String())
	}
}

func TestIncompleteCommand(t *testing.T) {
	cli := NewCLI()

	err := cli.Register("show version", "Show version information", func(w io.Writer) error {
		fmt.Fprintf(w, "Version 1.0.0\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	w := &strings.Builder{}
	cli.runLine("show", w)

	if w.String() != "% Command incomplete: show\n" {
		t.Fatalf("Unexpected output: %s", w.String())
	}
}

func TestUnknownCommand(t *testing.T) {
	cli := NewCLI()

	err := cli.Register("show version", "Show version information", func(w io.Writer) error {
		fmt.Fprintf(w, "Version 1.0.0\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	w := &strings.Builder{}
	cli.runLine("show foo", w)

	if w.String() != "% Unknown command: show foo\n" {
		t.Fatalf("Unexpected output: %s", w.String())
	}
}

func TestAmbiguousCommand(t *testing.T) {
	cli := NewCLI()

	err := cli.Register("show version", "Show version information", func(w io.Writer) error {
		fmt.Fprintf(w, "Version 1.0.0\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = cli.Register("show velocity", "Show velocity information", func(w io.Writer) error {
		fmt.Fprintf(w, "Velocity information\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	w := &strings.Builder{}
	cli.runLine("sh v", w)

	if w.String() != "% Ambiguous command: sh v\n" {
		t.Fatalf("Unexpected output: %s", w.String())
	}
}

func TestCommandWithArg(t *testing.T) {
	cli := NewCLI()

	err := cli.Register("show ip route A.B.C.D", "Show route to A.B.C.D", func(w io.Writer, addr netip.Addr) error {
		fmt.Fprintf(w, "Route to %s\n", addr)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	w := &strings.Builder{}
	cli.runLine("show ip route 1.1.1.1", w)

	if w.String() != "Route to 1.1.1.1\n" {
		t.Fatalf("Unexpected output: %s", w.String())
	}
}

func TestCommandWithArgPrefixMatching(t *testing.T) {
	cli := NewCLI()

	err := cli.Register("show ip route A.B.C.D", "Show route to A.B.C.D", func(w io.Writer, addr netip.Addr) error {
		fmt.Fprintf(w, "Route to %s\n", addr)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	w := &strings.Builder{}
	cli.runLine("sh ip ro 1.1.1.1", w)

	if w.String() != "Route to 1.1.1.1\n" {
		t.Fatalf("Unexpected output: %s", w.String())
	}
}

func TestCommandWithArgChoiceSameType(t *testing.T) {
	cli := NewCLI()

	err := cli.Register("show ip route <A.B.C.D|X:X:X::X>", "Show route to A.B.C.D", func(w io.Writer, addr netip.Addr) error {
		fmt.Fprintf(w, "Route to %s\n", addr)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	w := &strings.Builder{}
	cli.runLine("show ip route 1.1.1.1", w)

	if w.String() != "Route to 1.1.1.1\n" {
		t.Fatalf("Unexpected output: %s", w.String())
	}

	w = &strings.Builder{}
	cli.runLine("show ip route 1::1", w)

	if w.String() != "Route to 1::1\n" {
		t.Fatalf("Unexpected output: %s", w.String())
	}
}

func TestCommandWithArgChoiceDifferentTypes(t *testing.T) {
	cli := NewCLI()

	err := cli.Register("show ip route <A.B.C.D|all>", "Show route to A.B.C.D", func(w io.Writer, addr netip.Addr, all bool) error {
		if all {
			fmt.Fprintf(w, "All routes\n")
		} else {
			fmt.Fprintf(w, "Route to %s\n", addr)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	w := &strings.Builder{}
	cli.runLine("show ip route 1.1.1.1", w)

	if w.String() != "Route to 1.1.1.1\n" {
		t.Fatalf("Unexpected output: %s", w.String())
	}

	w = &strings.Builder{}
	cli.runLine("show ip route all", w)

	if w.String() != "All routes\n" {
		t.Fatalf("Unexpected output: %s", w.String())
	}
}

func TestDisambiguationBasedOnLaterTokens(t *testing.T) {
	cli := NewCLI()

	err := cli.Register("show ip bgp", "Show BGP information", func(w io.Writer) error {
		fmt.Fprintf(w, "BGP information\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = cli.Register("show interface brief", "Show interface brief", func(w io.Writer) error {
		fmt.Fprintf(w, "Interface brief\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	w := &strings.Builder{}
	cli.runLine("s i b", w)

	if w.String() != "% Ambiguous command: s i b\n" {
		t.Fatalf("Unexpected output: %s", w.String())
	}

	w = &strings.Builder{}
	cli.runLine("s i br", w)

	if w.String() != "Interface brief\n" {
		t.Fatalf("Unexpected output: %s", w.String())
	}

	w = &strings.Builder{}
	cli.runLine("s i bg", w)

	if w.String() != "BGP information\n" {
		t.Fatalf("Unexpected output: %s", w.String())
	}
}

func TestDisambiguationBasedOnLaterTokensWithParams(t *testing.T) {
	cli := NewCLI()

	err := cli.Register("show ip route A.B.C.D", "Show route to A.B.C.D", func(w io.Writer, addr netip.Addr) error {
		fmt.Fprintf(w, "IPv4 route to %s\n", addr)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	err = cli.Register("show ipv6 route X:X:X::X", "Show route to X:X:X::X", func(w io.Writer, addr netip.Addr) error {
		fmt.Fprintf(w, "IPv6 route to %s\n", addr)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	w := &strings.Builder{}
	cli.runLine("s i r 1.1.1.1", w)

	if w.String() != "IPv4 route to 1.1.1.1\n" {
		t.Fatalf("Unexpected output: %s", w.String())
	}
}

func TestRegistrationWrongArgs(t *testing.T) {
	cli := NewCLI()

	err := cli.Register("show ip route <A.B.C.D|all>", "Show route to A.B.C.D", func(w io.Writer, addr netip.Addr) error {
		fmt.Fprintf(w, "shouldn't register\n")
		return nil
	})

	if err == nil {
		t.Fatal("Expected error")
	}

	if !strings.Contains(err.Error(), "expected func(io.Writer, netip.Addr, bool) error, got func(io.Writer, netip.Addr) error") {
		t.Fatalf("Unexpected error: %s", err)
	}
}
