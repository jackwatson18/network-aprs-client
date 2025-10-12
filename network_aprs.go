package main

import (
	"flag"
	"fmt"
	"internal/AX25"
	"log"
	"net"

	"github.com/fatih/color"
)

func ListenOnlyKISS(server string) (<-chan string, <-chan error, error) {
	conn, err := net.Dial("tcp", server)
	if err != nil {
		return nil, nil, fmt.Errorf("error connecting to modem: %w", err)
	}

	tnc2_out := make(chan string)
	err_out := make(chan error)

	go func() {
		defer conn.Close()
		defer close(tnc2_out)
		defer close(err_out)
		buffer := make([]byte, 1024)

		for {
			mLen, err := conn.Read(buffer)

			if err != nil {
				err_out <- fmt.Errorf("error reading conn: %s", err)
				break
			}

			trimmed_bytes, err := AX25.StripKISSWrapper(buffer[:mLen])
			if err != nil {
				err_out <- fmt.Errorf("%v", err)
				continue
			}
			frame_struct, err := AX25.ConvertBytesToAX25(trimmed_bytes)
			if err != nil {
				err_out <- fmt.Errorf("%v", err)
				continue
			}

			tnc2_out <- fmt.Sprintf("%v\n", frame_struct.TNC2())
		}
	}()

	return tnc2_out, err_out, nil
}

func ConsolePrinter(str <-chan string, err <-chan error) {
	for {
		select {
		case err, ok := <-err:
			if !ok {
				return
			}
			c := color.New(color.FgRed).Add(color.Bold)
			c.Printf("%s\n", err)

		case tnc2, ok := <-str:
			if !ok {
				return
			}
			c := color.New(color.FgGreen).Add(color.Bold)
			c.Printf("%s\n", tnc2)
		}
	}
}

func main() {

	serverPtr := flag.String("srv", "localhost:8001", "KISS Server Address")
	flag.Parse()
	fmt.Println(*serverPtr)

	tnc2_out, err_out, err := ListenOnlyKISS(*serverPtr)
	if err != nil {
		log.Fatalf("Failed to connect to server: %s", err)
	}

	ConsolePrinter(tnc2_out, err_out)

}
