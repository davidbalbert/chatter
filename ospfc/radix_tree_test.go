package main

import (
	"testing"
)

func TestLoad(t *testing.T) {
	n := &node{}
	n.store("foo", 1)
	n.store("foobar", 2)
	n.store("bar", 3)

	value, ok := n.load("foo")
	if !ok {
		t.Fatal("expected foo to be found")
	} else if value != 1 {
		t.Fatalf("expected foo to be 1, got %d", value)
	}

	value, ok = n.load("foobar")
	if !ok {
		t.Fatal("expected foobar to be found")
	} else if value != 2 {
		t.Fatalf("expected foobar to be 2, got %d", value)
	}

	value, ok = n.load("bar")
	if !ok {
		t.Fatal("expected bar to be found")
	} else if value != 3 {
		t.Fatalf("expected bar to be 3, got %d", value)
	}
}

func TestWalkBytes(t *testing.T) {
	n := &node{}
	n.store("foo", 1)
	n.store("foobar", 2)
	n.store("bar", 3)

	var prefixes []string

	err := n.walkBytes("", func(prefix string) error {
		prefixes = append(prefixes, prefix)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"", "b", "ba", "bar", "f", "fo", "foo", "foob", "fooba", "foobar"}

	if len(prefixes) != len(expected) {
		t.Fatalf("expected %d prefixes, got %d", len(expected), len(prefixes))
	}

	for i, prefix := range prefixes {
		if prefix != expected[i] {
			t.Fatalf("expected prefixes %#v, got %#v", expected, prefixes)
		}
	}
}

func TestWalkBytesSkipPrefix(t *testing.T) {
	n := &node{}
	n.store("foo", 1)
	n.store("foobar", 2)
	n.store("bar", 3)

	var prefixes []string

	err := n.walkBytes("", func(prefix string) error {
		prefixes = append(prefixes, prefix)

		if prefix == "b" {
			return errSkipPrefix
		}

		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"", "b", "f", "fo", "foo", "foob", "fooba", "foobar"}

	if len(prefixes) != len(expected) {
		t.Fatalf("expected %d prefixes, got %d", len(expected), len(prefixes))
	}

	for i, prefix := range prefixes {
		if prefix != expected[i] {
			t.Fatalf("expected prefixes %#v, got %#v", expected, prefixes)
		}
	}
}

func TestWalkBytesSkipAll(t *testing.T) {
	n := &node{}
	n.store("foo", 1)
	n.store("foobar", 2)
	n.store("bar", 3)

	var prefixes []string

	err := n.walkBytes("", func(prefix string) error {
		prefixes = append(prefixes, prefix)

		if prefix == "b" {
			return errSkipAll
		}

		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"", "b"}

	if len(prefixes) != len(expected) {
		t.Fatalf("expected %d prefixes, got %d", len(expected), len(prefixes))
	}

	for i, prefix := range prefixes {
		if prefix != expected[i] {
			t.Fatalf("expected prefixes %#v, got %#v", expected, prefixes)
		}
	}
}

func TestWalkBytesWithRootBeforeBranch(t *testing.T) {
	n := &node{}
	n.store("foo", 1)
	n.store("foobar", 2)
	n.store("bar", 3)

	var prefixes []string

	err := n.walkBytes("b", func(prefix string) error {
		prefixes = append(prefixes, prefix)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"b", "ba", "bar"}

	if len(prefixes) != len(expected) {
		t.Fatalf("expected %d prefixes, got %d", len(expected), len(prefixes))
	}

	for i, prefix := range prefixes {
		if prefix != expected[i] {
			t.Fatalf("expected prefixes %#v, got %#v", expected, prefixes)
		}
	}
}

func TestWalkBytesWithRootAtBranch(t *testing.T) {
	n := &node{}
	n.store("foo", 1)
	n.store("foobar", 2)
	n.store("bar", 3)

	var prefixes []string

	err := n.walkBytes("foo", func(prefix string) error {
		prefixes = append(prefixes, prefix)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"foo", "foob", "fooba", "foobar"}

	if len(prefixes) != len(expected) {
		t.Fatalf("expected %d prefixes, got %d", len(expected), len(prefixes))
	}

	for i, prefix := range prefixes {
		if prefix != expected[i] {
			t.Fatalf("expected prefixes %#v, got %#v", expected, prefixes)
		}
	}
}

func TestWalkBytesWithRootAfterBranch(t *testing.T) {
	n := &node{}
	n.store("foo", 1)
	n.store("foobar", 2)
	n.store("bar", 3)

	var prefixes []string

	err := n.walkBytes("foob", func(prefix string) error {
		prefixes = append(prefixes, prefix)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"foob", "fooba", "foobar"}

	if len(prefixes) != len(expected) {
		t.Fatalf("expected %d prefixes, got %d", len(expected), len(prefixes))
	}

	for i, prefix := range prefixes {
		if prefix != expected[i] {
			t.Fatalf("expected prefixes %#v, got %#v", expected, prefixes)
		}
	}
}

func TestWalkBytesWithNonexistentRoot(t *testing.T) {
	n := &node{}
	n.store("foo", 1)
	n.store("foobar", 2)
	n.store("bar", 3)

	var prefixes []string

	err := n.walkBytes("fob", func(prefix string) error {
		prefixes = append(prefixes, prefix)
		return nil
	})

	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if len(prefixes) != 0 {
		t.Fatalf("expected no prefixes, got %#v", prefixes)
	}
}
