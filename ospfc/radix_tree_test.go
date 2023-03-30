package main

import (
	"reflect"
	"testing"
)

func TestStoreAndLoad(t *testing.T) {
	n := &node{}
	n.store("foobar", 1)
	n.store("foo", 2)
	n.store("bar", 3)

	value, ok := n.load("foobar")
	if !ok {
		t.Fatal("expected foobar to be found")
	} else if value != 1 {
		t.Fatalf("expected foobar to be 1, got %d", value)
	}

	value, ok = n.load("foo")
	if !ok {
		t.Fatal("expected foo to be found")
	} else if value != 2 {
		t.Fatalf("expected foo to be 2, got %d", value)
	}

	value, ok = n.load("bar")
	if !ok {
		t.Fatal("expected bar to be found")
	} else if value != 3 {
		t.Fatalf("expected bar to be 3, got %d", value)
	}
}

func TestLoadNotFound(t *testing.T) {
	n := &node{}
	n.store("foobar", 1)
	n.store("foo", 2)

	_, ok := n.load("fo")
	if ok {
		t.Fatal("expected fo to not be found")
	}

	_, ok = n.load("foob")
	if ok {
		t.Fatal("expected foob to not be found")
	}

	_, ok = n.load("bar")
	if ok {
		t.Fatal("expected bar to not be found")
	}
}

func TestStoreAndLoadEmptyKey(t *testing.T) {
	n := &node{}
	n.store("", 1)

	value, ok := n.load("")
	if !ok {
		t.Fatal("expected empty key to be found")
	} else if value != 1 {
		t.Fatalf("expected empty key to be 1, got %d", value)
	}
}

func TestNonExistantEmptyKeyLoad(t *testing.T) {
	n := &node{}

	_, ok := n.load("")
	if ok {
		t.Fatal("expected empty key to not be found")
	}

	n.store("foo", 1)

	_, ok = n.load("")
	if ok {
		t.Fatal("expected empty key to not be found")
	}
}

func TestOverwrite(t *testing.T) {
	n := &node{}
	n.store("foo", 1)

	value, ok := n.load("foo")
	if !ok {
		t.Fatal("expected foo to be found")
	} else if value != 1 {
		t.Fatalf("expected foo to be 1, got %d", value)
	}

	n.store("foo", 2)

	value, ok = n.load("foo")
	if !ok {
		t.Fatal("expected foo to be found")
	} else if value != 2 {
		t.Fatalf("expected foo to be 2, got %d", value)
	}
}

func TestWalkPartialTokensExactMatch(t *testing.T) {
	n := &node{}
	n.store("show version", 1)
	n.store("show version detail", 2)
	n.store("show name", 3)
	n.store("show number", 4)

	var keys []string
	var values []int
	n.walkPartialTokens("show version", ' ', func(prefix string, value any) error {
		keys = append(keys, prefix)
		values = append(values, value.(int))
		return nil
	})

	expectedKeys := []string{"show version"}
	expectedValues := []int{1}

	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Fatalf("expected prefixes %#v, got %#v", expectedKeys, keys)
	}

	if !reflect.DeepEqual(values, expectedValues) {
		t.Fatalf("expected values %#v, got %#v", expectedValues, values)
	}
}

func TestWalkPartialTokensPrefixMatch(t *testing.T) {
	n := &node{}
	n.store("show version", 1)
	n.store("show version detail", 2)
	n.store("show name", 3)
	n.store("show number", 4)

	var keys []string
	var values []int
	n.walkPartialTokens("sh ver", ' ', func(prefix string, value any) error {
		keys = append(keys, prefix)
		values = append(values, value.(int))
		return nil
	})

	expectedKeys := []string{"show version"}
	expectedValues := []int{1}

	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Fatalf("expected prefixes %#v, got %#v", expectedKeys, keys)
	}

	if !reflect.DeepEqual(values, expectedValues) {
		t.Fatalf("expected values %#v, got %#v", expectedValues, values)
	}
}

func TestWalkPartialTokensMultipleMatches(t *testing.T) {
	n := &node{}
	n.store("show version", 1)
	n.store("show version detail", 2)
	n.store("show name", 3)
	n.store("show number", 4)

	var keys []string
	var values []int
	n.walkPartialTokens("sh n", ' ', func(prefix string, value any) error {
		keys = append(keys, prefix)
		values = append(values, value.(int))
		return nil
	})

	expectedKeys := []string{"show name", "show number"}
	expectedValues := []int{3, 4}

	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Fatalf("expected prefixes %#v, got %#v", expectedKeys, keys)
	}

	if !reflect.DeepEqual(values, expectedValues) {
		t.Fatalf("expected values %#v, got %#v", expectedValues, values)
	}

	keys = nil
	values = nil
	n.walkPartialTokens("sh na", ' ', func(prefix string, value any) error {
		keys = append(keys, prefix)
		values = append(values, value.(int))
		return nil
	})

	expectedKeys = []string{"show name"}
	expectedValues = []int{3}

	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Fatalf("expected prefixes %#v, got %#v", expectedKeys, keys)
	}

	if !reflect.DeepEqual(values, expectedValues) {
		t.Fatalf("expected values %#v, got %#v", expectedValues, values)
	}

	keys = nil
	values = nil
	n.walkPartialTokens("sh nu", ' ', func(prefix string, value any) error {
		keys = append(keys, prefix)
		values = append(values, value.(int))
		return nil
	})

	expectedKeys = []string{"show number"}
	expectedValues = []int{4}

	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Fatalf("expected prefixes %#v, got %#v", expectedKeys, keys)
	}

	if !reflect.DeepEqual(values, expectedValues) {
		t.Fatalf("expected values %#v, got %#v", expectedValues, values)
	}
}

func TestWalkPartialTokensEdgeDoesntMatch(t *testing.T) {
	n := &node{}
	n.store("show version", 1)
	n.store("show version detail", 2)
	n.store("show name", 3)
	n.store("show number", 4)

	var keys []string
	var values []int

	n.walkPartialTokens("shaw", ' ', func(prefix string, value any) error {
		keys = append(keys, prefix)
		values = append(values, value.(int))
		return nil
	})

	expectedKeys := []string(nil)
	expectedValues := []int(nil)

	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Fatalf("expected prefixes %#v, got %#v", expectedKeys, keys)
	}

	if !reflect.DeepEqual(values, expectedValues) {
		t.Fatalf("expected values %#v, got %#v", expectedValues, values)
	}
}

func TestWalkPartialTokensTooFewTokens(t *testing.T) {
	n := &node{}
	n.store("show version", 1)
	n.store("show version detail", 2)
	n.store("show name", 3)
	n.store("show number", 4)

	var keys []string
	var values []int

	n.walkPartialTokens("sh", ' ', func(prefix string, value any) error {
		keys = append(keys, prefix)
		values = append(values, value.(int))
		return nil
	})

	expectedKeys := []string(nil)
	expectedValues := []int(nil)

	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Fatalf("expected prefixes %#v, got %#v", expectedKeys, keys)
	}

	if !reflect.DeepEqual(values, expectedValues) {
		t.Fatalf("expected values %#v, got %#v", expectedValues, values)
	}
}

func TestWalkPartialTokensTooMany(t *testing.T) {
	n := &node{}
	n.store("show version", 1)
	n.store("show version detail", 2)
	n.store("show name", 3)
	n.store("show number", 4)

	var keys []string
	var values []int

	n.walkPartialTokens("sh na foo", ' ', func(prefix string, value any) error {
		keys = append(keys, prefix)
		values = append(values, value.(int))
		return nil
	})

	expectedKeys := []string(nil)
	expectedValues := []int(nil)

	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Fatalf("expected prefixes %#v, got %#v", expectedKeys, keys)
	}

	if !reflect.DeepEqual(values, expectedValues) {
		t.Fatalf("expected values %#v, got %#v", expectedValues, values)
	}
}

func TestWalkPartialTokensEmptyQuery(t *testing.T) {
	n := &node{}
	n.store("", 1)

	var keys []string
	var values []int

	n.walkPartialTokens("", ' ', func(prefix string, value any) error {
		keys = append(keys, prefix)
		values = append(values, value.(int))
		return nil
	})

	expectedKeys := []string{""}
	expectedValues := []int{1}

	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Fatalf("expected prefixes %#v, got %#v", expectedKeys, keys)
	}

	if !reflect.DeepEqual(values, expectedValues) {
		t.Fatalf("expected values %#v, got %#v", expectedValues, values)
	}
}

func TestWalkPartialTokensEmptyQueryNoMatch(t *testing.T) {
	n := &node{}
	n.store("show version", 1)

	var keys []string
	var values []int

	n.walkPartialTokens("", ' ', func(prefix string, value any) error {
		keys = append(keys, prefix)
		values = append(values, value.(int))
		return nil
	})

	expectedKeys := []string(nil)
	expectedValues := []int(nil)

	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Fatalf("expected prefixes %#v, got %#v", expectedKeys, keys)
	}

	if !reflect.DeepEqual(values, expectedValues) {
		t.Fatalf("expected values %#v, got %#v", expectedValues, values)
	}
}

func TestWalkPartialTokensEmptyTree(t *testing.T) {
	n := &node{}

	var keys []string
	var values []int

	n.walkPartialTokens("sh ver", ' ', func(prefix string, value any) error {
		keys = append(keys, prefix)
		values = append(values, value.(int))
		return nil
	})

	expectedKeys := []string(nil)
	expectedValues := []int(nil)

	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Fatalf("expected prefixes %#v, got %#v", expectedKeys, keys)
	}

	if !reflect.DeepEqual(values, expectedValues) {
		t.Fatalf("expected values %#v, got %#v", expectedValues, values)
	}
}

func TestWalkPartialTokensSeparatorWithinEdge(t *testing.T) {
	n := &node{}

	n.store("foo bar baz", 1)
	n.store("foo bar", 2)

	var keys []string
	var values []int

	n.walkPartialTokens("foo bar", ' ', func(prefix string, value any) error {
		keys = append(keys, prefix)
		values = append(values, value.(int))
		return nil
	})

	expectedKeys := []string{"foo bar"}
	expectedValues := []int{2}

	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Fatalf("expected prefixes %#v, got %#v", expectedKeys, keys)
	}

	if !reflect.DeepEqual(values, expectedValues) {
		t.Fatalf("expected values %#v, got %#v", expectedValues, values)
	}
}
