package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/davidbalbert/ospfd/vty"
)

func ReadLine(prompt string) (string, error) {
	fmt.Print(prompt)
	scanner := bufio.NewReader(os.Stdin)
	line, err := scanner.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(line), nil
}

func main() {
	client, err := vty.Dial()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer client.Close()

	for {
		req, err := ReadLine("ospfd# ")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if req == "" {
			continue
		}

		if req == "exit" {
			break
		}

		resp, err := client.Send(req)
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println(resp)
	}
}
