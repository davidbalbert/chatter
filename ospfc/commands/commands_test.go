package commands

import (
	"testing"
)

func TestMergeLiterals(t *testing.T) {
	l1 := &literal{value: "foo"}
	l2 := &literal{value: "bar"}

	merged := l1.Merge(l2)

	_, ok := merged.(*fork)
	if !ok {
		t.Fatalf("expected *fork, got %T", merged)
	}

	// spec := `
	// 	fork[
	// 		literal:foo[
	// 			join.1
	// 		],
	// 		literal:bar[
	// 			join.1
	// 		]
	// 	]
	// `

	c := merged.Children()

	if len(c) != 2 {
		t.Fatalf("expected 2 children, got %d", len(c))
	}

	if c[0].Name() != "literal:foo" {
		t.Fatalf("expected child 0 to be literal:foo, got %q", c[0].Name())
	}

	if c[1].Name() != "literal:bar" {
		t.Fatalf("expected child 1 to be literal:bar, got %q", c[1].Name())
	}

	if len(c[0].Children()) != 1 {
		t.Fatalf("expected literal:foo to have 1 child, got %d", len(c[0].Children()))
	}

	if len(c[1].Children()) != 1 {
		t.Fatalf("expected literal:bar to have 1 child, got %d", len(c[1].Children()))
	}

	j1, ok := c[0].Children()[0].(*join)
	if !ok {
		t.Fatalf("expected literal:foo to have a join child, got %T", c[0].Children()[0])
	}

	j2, ok := c[1].Children()[1].(*join)
	if !ok {
		t.Fatalf("expected literal:bar to have a join child, got %T", c[1].Children()[0])
	}

	if j1.child != j2.child {
		t.Fatalf("expected join children to be the same, got %q and %q", j1.child.Name(), j2.child.Name())
	}
}
