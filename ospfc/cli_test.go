package main

import (
	"reflect"
	"testing"
)

func TestExpandAndLoad(t *testing.T) {
	c := &cli{}
	c.register("show version", 1)
	c.register("show version detail", 2)
	c.register("show name", 3)
	c.register("show number", 4)

	keys, _, err := c.expandAndLoad("show version")
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"show version"}
	if !reflect.DeepEqual(keys, expected) {
		t.Fatalf("got %#v, expected %#v", keys, expected)
	}

	keys, _, err = c.expandAndLoad("sh ver")
	if err != nil {
		t.Fatal(err)
	}

	expected = []string{"show version"}
	if !reflect.DeepEqual(keys, expected) {
		t.Fatalf("got %#v, expected %#v", keys, expected)
	}

	keys, _, err = c.expandAndLoad("s v d")
	if err != nil {
		t.Fatal(err)
	}

	expected = []string{"show version detail"}
	if !reflect.DeepEqual(keys, expected) {
		t.Fatalf("got %#v, expected %#v", keys, expected)
	}

	keys, _, err = c.expandAndLoad("sh n")
	if err != nil {
		t.Fatal(err)
	}

	expected = []string{"show name", "show number"}
	if !reflect.DeepEqual(keys, expected) {
		t.Fatalf("got %#v, expected %#v", keys, expected)
	}

	keys, _, err = c.expandAndLoad("sh na")
	if err != nil {
		t.Fatal(err)
	}

	expected = []string{"show name"}
	if !reflect.DeepEqual(keys, expected) {
		t.Fatalf("got %#v, expected %#v", keys, expected)
	}

	keys, _, err = c.expandAndLoad("sh nu")
	if err != nil {
		t.Fatal(err)
	}

	expected = []string{"show number"}
	if !reflect.DeepEqual(keys, expected) {
		t.Fatalf("got %#v, expected %#v", keys, expected)
	}

	keys, _, err = c.expandAndLoad("sh nu foo")
	if err != nil {
		t.Fatal(err)
	}

	expected = []string(nil)
	if !reflect.DeepEqual(keys, expected) {
		t.Fatalf("got %#v, expected %#v", keys, expected)
	}
}

func TestExpandAndLoadWithValue(t *testing.T) {
	c := &cli{}
	c.register("show version", 1)
	c.register("show version detail", 2)
	c.register("show name", 3)
	c.register("show number", 4)

	keys, values, err := c.expandAndLoad("show version")
	if err != nil {
		t.Fatal(err)
	}

	expectedKeys := []string{"show version"}
	expectedValues := []any{1}
	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Fatalf("got %#v, expected %#v", keys, expectedKeys)
	}

	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Fatalf("got %#v, expected %#v", keys, expectedKeys)
	}

	if !reflect.DeepEqual(values, expectedValues) {
		t.Fatalf("got %#v, expected %#v", values, expectedValues)
	}

	keys, values, err = c.expandAndLoad("sh ver")
	if err != nil {
		t.Fatal(err)
	}

	expectedKeys = []string{"show version"}
	expectedValues = []any{1}

	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Fatalf("got %#v, expected %#v", keys, expectedKeys)
	}

	if !reflect.DeepEqual(values, expectedValues) {
		t.Fatalf("got %#v, expected %#v", values, expectedValues)
	}
}
