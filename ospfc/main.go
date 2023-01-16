package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/davidbalbert/ospfd/vty"
)

func ReadLine() (string, error) {
	scanner := bufio.NewReader(os.Stdin)
	line, err := scanner.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(line), nil
}

func main() {
	conn, err := vty.Dial()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer conn.Close()

	stdin := make(chan string)
	socket := make(chan string)
	done := make(chan struct{})

	go func() {
		for {
			line, err := ReadLine()
			if err != nil {
				select {
				case <-done:
					return
				default:
				}

				fmt.Println(err)
				close(done)
				return
			}

			if line == "" {
				continue
			}

			stdin <- line
		}
	}()

	go func() {
		for {
			frame, err := conn.ReadFrame()
			if err != nil {
				select {
				case <-done:
					return
				default:
				}

				if err != io.EOF {
					fmt.Println(err)
				}

				close(done)
				return
			}

			socket <- string(frame)
		}
	}()

	for {
		fmt.Print("ospfd# ")
		select {
		case req := <-stdin:
			if req == "exit" {
				close(done)
				return
			}

			err := conn.WriteFrame([]byte(req))
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			select {
			case resp := <-socket:
				fmt.Println(resp)
			case <-done:
				return
			}
		case <-done:
			return
		}
	}
}
