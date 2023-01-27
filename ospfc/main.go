package main

import (
	"context"
	"fmt"
	"os"

	"github.com/davidbalbert/ospfd/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	target := fmt.Sprintf("unix://%s", api.SocketPath)

	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to dial: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	client := api.NewAPIClient(conn)

	resp, err := client.GetRandInt(context.Background(), &api.Empty{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get rand int: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(resp.Value)

	resp2, err := client.GetRandString(context.Background(), &api.Empty{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get rand string: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(resp2.Value)
}

// func readLine() (string, error) {
// 	reader := bufio.NewReader(os.Stdin)
// 	line, err := reader.ReadString('\n')
// 	if err != nil {
// 		return "", err
// 	}

// 	return strings.TrimSpace(line), nil
// }

// func main() {
// 	conn, err := vty.Dial()
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}
// 	defer conn.Close()

// 	stdin := make(chan string)
// 	socket := make(chan string)
// 	done := make(chan struct{})

// 	go func() {
// 		for {
// 			line, err := readLine()
// 			if err != nil {
// 				select {
// 				case <-done:
// 					return
// 				default:
// 				}

// 				fmt.Println(err)
// 				close(done)
// 				return
// 			}

// 			if line == "" {
// 				continue
// 			}

// 			stdin <- line
// 		}
// 	}()

// 	go func() {
// 		for {
// 			frame, err := conn.ReadFrame()
// 			if err != nil {
// 				select {
// 				case <-done:
// 					return
// 				default:
// 				}

// 				if err != io.EOF {
// 					fmt.Println(err)
// 				}

// 				close(done)
// 				return
// 			}

// 			socket <- string(frame)
// 		}
// 	}()

// 	for {
// 		fmt.Print("ospfd# ")
// 		select {
// 		case req := <-stdin:
// 			if req == "exit" {
// 				close(done)
// 				return
// 			}

// 			err := conn.WriteFrame([]byte(req))
// 			if err != nil {
// 				fmt.Println(err)
// 				os.Exit(1)
// 			}

// 			select {
// 			case resp := <-socket:
// 				fmt.Println(resp)
// 			case <-done:
// 				return
// 			}
// 		case <-done:
// 			return
// 		}
// 	}
// }
