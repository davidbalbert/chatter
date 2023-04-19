package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davidbalbert/chatter/chatterc/commands"
	"golang.org/x/term"
)

func commonPrefixLen(ss ...string) int {
	if len(ss) == 0 {
		return 0
	}

	prefixLen := len(ss[0])
	for _, s := range ss[1:] {
		i := 0
		for ; i < len(s) && i < prefixLen; i++ {
			if s[i] != ss[0][i] {
				break
			}
		}
		prefixLen = i
	}

	return prefixLen
}

type fder interface {
	Fd() uintptr
}

type writeFder interface {
	io.Writer
	Fd() uintptr
}

type readWriteFder interface {
	io.ReadWriter
	Fd() uintptr
}

type CLI struct {
	running bool
	root    *commands.Node
	prompt  string
	lastKey rune
}

func NewCLI() *CLI {
	cli := &CLI{prompt: "chatterc# "}

	cli.MustRegister("exit", "Exit the CLI", func(w io.Writer) error {
		cli.running = false
		return nil
	})

	cli.MustRegister("quit", "Exit the CLI", func(w io.Writer) error {
		cli.running = false
		return nil
	})

	return cli
}

// A generic function Tabulate that takes a list of rows of type T (any),
// a list of strings (the headers), and a function that takes a row and
// returns a list of strings (the columns). It returns a string that
// contains the tabulated data.
func tabulate[T any](items []T, headers []string, f func(T) []string) ([]string, error) {
	// Get the column widths
	columnWidths := make([]int, len(headers))
	for i, h := range headers {
		columnWidths[i] = len(h)
	}

	cells := make([][]string, len(items))

	for i, item := range items {
		cells[i] = f(item)

		if len(cells[i]) != len(headers) {
			return nil, fmt.Errorf("invalid number of columns for item %d", i)
		}

		for j, cell := range cells[i] {
			if len(cell) > columnWidths[j] {
				columnWidths[j] = len(cell)
			}
		}
	}

	// Build the table
	table := make([]string, len(items)+2)

	// Header
	header := ""
	for i, h := range headers {
		header += fmt.Sprintf("%-*s", columnWidths[i]+3, h)
	}

	table[0] = header

	// Separator
	separator := ""
	for i := range headers {
		separator += fmt.Sprintf("%-*s", columnWidths[i]+3, strings.Repeat("-", columnWidths[i]))
	}

	table[1] = separator

	// Rows
	for i, row := range cells {
		table[i+2] = ""
		for j, cell := range row {
			table[i+2] += fmt.Sprintf("%-*s", columnWidths[j]+3, cell)
		}
	}

	return table, nil
}

func wrap(f fder, indent int, words []string) []string {
	width, _, err := term.GetSize(int(f.Fd()))
	if err != nil {
		return words
	}

	width -= indent

	longestLen := 0
	for _, w := range words {
		if len(w) > longestLen {
			longestLen = len(w)
		}
	}

	perRow := width / (longestLen + 2)

	if perRow == 0 {
		return words
	}

	rows := len(words) / perRow
	if len(words)%perRow != 0 {
		rows++
	}

	lines := make([]string, rows)
	for i := 0; i < rows; i++ {
		lines[i] = strings.Repeat(" ", indent)

		for j := 0; j < perRow; j++ {
			index := i + j*rows
			if index >= len(words) {
				break
			}

			lines[i] += words[index]
			if j != perRow-1 {
				lines[i] += strings.Repeat(" ", longestLen-len(words[index])+2)
			}
		}
	}

	return lines
}

func (cli *CLI) autocompleteWithTab(w writeFder, line string, pos int) (newLine string, newPos int, ok bool) {
	prefix := line[:pos]
	rest := line[pos:]

	options, offset, err := cli.root.GetAutocompleteOptions(prefix)
	if err != nil {
		fmt.Fprintf(w, "%s%s\n", cli.prompt, line)
		fmt.Fprintf(w, "%% Error getting autocomplete options: %v\n", err)
		return "", 0, false
	}

	if len(options) == 0 {
		fmt.Fprintf(w, "\a")
		return "", 0, false
	} else if len(options) == 1 {
		new := prefix + options[0][offset:]

		if !strings.HasPrefix(rest, " ") {
			new += " "
		}

		return new + rest, len(new), true
	} else if cli.lastKey != '\t' {
		prefixLen := commonPrefixLen(options...)
		new := prefix + options[0][offset:prefixLen]

		fmt.Fprintf(w, "\a")

		return new + rest, len(new), true
	} else {
		fmt.Fprintf(w, "%s%s\n", cli.prompt, line)

		for _, l := range wrap(w, 0, options) {
			fmt.Fprintf(w, "%s\n", l)
		}

		return "", 0, false
	}
}

type NodeSlice []*commands.Node

func (ns NodeSlice) Len() int {
	return len(ns)
}

func (ns NodeSlice) Less(i, j int) bool {
	return ns[i].String() < ns[j].String()
}

func (ns NodeSlice) Swap(i, j int) {
	ns[i], ns[j] = ns[j], ns[i]
}

func (cli *CLI) autocompleteWithQuestionMark(w writeFder, line string, pos int) (newLine string, newPos int, ok bool) {
	nodes, err := cli.root.GetAutocompleteNodes(line)
	if err != nil {
		fmt.Fprintf(w, "%s%s\n", cli.prompt, line)
		fmt.Fprintf(w, "%% Error getting autocomplete nodes: %v\n", err)
		return line, pos, true
	}

	// Check to see if we should add <cr> as the first option. We do this if
	// we're at the beginning of a token (i.e. the last character is a space)
	// and the line matches a complete command.
	var cr bool
	lastRune, _ := utf8.DecodeLastRuneInString(line)
	if unicode.IsSpace(lastRune) {
		matches := cli.root.Match(line)

		if len(matches) == 1 && matches[0].IsComplete() {
			cr = true
		}
	}

	if len(nodes) == 0 && !cr {
		fmt.Fprintf(w, "%s%s\n", cli.prompt, line)
		fmt.Fprintf(w, "%% There is no matched command.\n")
		return line, pos, true
	}

	longestTokenLen := 0
	for _, n := range nodes {
		if len(n.String()) > longestTokenLen {
			longestTokenLen = len(n.String())
		}
	}

	sort.Sort(NodeSlice(nodes))

	fmt.Fprintf(w, "%s%s\n", cli.prompt, line)

	if cr {
		fmt.Fprintf(w, "  <cr>\n")
	}

	for _, n := range nodes {
		description := n.Description()
		if description == "" {
			description = "Missing description"
		}

		fmt.Fprintf(w, "  %-*s  %s\n", longestTokenLen, n.String(), description)

		opts, err := n.OptionsFromAutocompleteFunc("")
		if err != nil {
			fmt.Fprintf(w, "%% Error getting options: %v\n", err)
			continue
		}

		for _, l := range wrap(w, 5, opts) {
			fmt.Fprintf(w, "%s\n", l)
		}
	}

	return line, pos, true
}

func (cli *CLI) autocomplete(w writeFder, line string, pos int, key rune) (newLine string, newPos int, ok bool) {
	defer func() {
		cli.lastKey = key
	}()

	if key == '\t' {
		return cli.autocompleteWithTab(w, line, pos)
	} else if key == '?' {
		return cli.autocompleteWithQuestionMark(w, line, pos)
	}

	return "", 0, false
}

type terminal struct {
	*term.Terminal
	fder
}

func (cli *CLI) runLine(line string, w io.Writer) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	matches := cli.root.Match(line)
	completeMatches := make([]*commands.Match, 0, len(matches))

	for _, m := range matches {
		if m.IsComplete() {
			completeMatches = append(completeMatches, m)
		}
	}

	if len(completeMatches) == 0 && len(matches) == 0 {
		fmt.Fprintf(w, "%% Unknown command: %s\n", line)
		return
	} else if len(completeMatches) == 0 {
		fmt.Fprintf(w, "%% Command incomplete: %s\n", line)
		return
	} else if len(completeMatches) > 1 || len(matches) > 1 {
		fmt.Fprintf(w, "%% Ambiguous command: %s\n", line)
		return
	}

	invoker, err := matches[0].Invoker()
	if err != nil {
		fmt.Fprintf(w, "%% Error running command: %v\n", err)
		return
	}

	err = invoker.Run(w)
	if err != nil {
		fmt.Fprintf(w, "%% Error running command: %v\n", err)
		return
	}
}

func (cli *CLI) Run(rw readWriteFder) {
	t := &terminal{term.NewTerminal(rw, cli.prompt), rw}

	autoCompleteCallback := func(line string, pos int, key rune) (newLine string, newPos int, ok bool) {
		return cli.autocomplete(t, line, pos, key)
	}

	t.AutoCompleteCallback = autoCompleteCallback

	cli.running = true

	for cli.running {
		line, err := t.ReadLine()
		if err == io.EOF {
			// Hack to get around the fact that when Terminal.Readline() gets a ^C (technically an
			// ETX character), it doesn't advance its buffer before returning io.EOF. This means that
			// every subsequent call to Readline() will return an empty string. To get around this, we
			// just create a new terminal, which has the effect of resetting the buffer.
			t = &terminal{term.NewTerminal(rw, cli.prompt), rw}
			t.AutoCompleteCallback = autoCompleteCallback

			fmt.Fprintln(t)
		} else if err != nil {
			fmt.Fprintf(t, "%% Error reading line: %v\n", err)
			break
		}

		cli.runLine(line, t)
	}
}

func (cli *CLI) Register(command string, description string, handlerFunc any) error {
	n, err := commands.ParseDeclaration(command)
	if err != nil {
		return err
	}

	for _, l := range n.Leaves() {
		err := l.SetHandlerFunc(handlerFunc)
		if err != nil {
			return err
		}

		err = l.SetDescription(description)
		if err != nil {
			return err
		}
	}

	newRoot, err := cli.root.Merge(n)
	if err != nil {
		return err
	}

	cli.root = newRoot

	return nil
}

func (cli *CLI) MustRegister(command string, description string, handlerFunc any) {
	err := cli.Register(command, description, handlerFunc)
	if err != nil {
		panic(err)
	}
}

func (cli *CLI) Document(command string, description string) error {
	n, err := commands.ParseDeclaration(command)
	if err != nil {
		return err
	}

	for _, l := range n.Leaves() {
		l.SetDescription(description)
	}

	newRoot, err := cli.root.MergeWithoutExplicitChoiceRestrictions(n)
	if err != nil {
		return err
	}

	cli.root = newRoot

	return nil
}

func (cli *CLI) MustDocument(command string, description string) {
	err := cli.Document(command, description)
	if err != nil {
		panic(err)
	}
}

func (cli *CLI) RegisterAutocomplete(command string, autocompleteFunc commands.AutocompleteFunc) error {
	n, err := commands.ParseDeclaration(command)
	if err != nil {
		return err
	}

	for _, l := range n.Leaves() {
		l.SetAutocompleteFunc(autocompleteFunc)
	}

	newRoot, err := cli.root.MergeWithoutExplicitChoiceRestrictions(n)
	if err != nil {
		return err
	}

	cli.root = newRoot

	return nil
}

func (cli *CLI) MustRegisterAutocomplete(command string, autocompleteFunc commands.AutocompleteFunc) {
	err := cli.RegisterAutocomplete(command, autocompleteFunc)
	if err != nil {
		panic(err)
	}
}
