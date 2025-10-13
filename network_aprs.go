package main

import (
	"flag"
	"fmt"
	"internal/AX25"
	"log"
	"net"
	"strings"
	"time"

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

func ListenOnlyAPRSIS() (<-chan string, <-chan error, error) {
	// Connect to an APRS-IS server (e.g., rotate.aprs2.net:14580 for a filtered feed)
	conn, err := net.DialTimeout("tcp", "rotate.aprs2.net:14580", 5*time.Second)
	if err != nil {
		return nil, nil, fmt.Errorf("error connecting: %w", err)
	}
	tnc2_out := make(chan string)
	err_out := make(chan error)
	buffer := make([]byte, 1024)

	_, err = conn.Write([]byte("user N0CALL pass -1 filter r/35.28/-120.66/100\n"))

	if err != nil {
		return nil, nil, fmt.Errorf("error authenticating: %w", err)
	}

	go func() {
		defer conn.Close()
		defer close(tnc2_out)
		defer close(err_out)
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				err_out <- fmt.Errorf("problem with APRS-IS socket: %w", err)
				return
			}
			data := string(buffer[:n])
			packets := strings.SplitSeq(data, "\r\n")
			for packet := range packets {
				if packet != "" && !strings.HasPrefix(packet, "#") {
					tnc2_out <- packet

				}
			}

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

	// tnc2_out, err_out, err := ListenOnlyKISS(*serverPtr)
	tnc2_out, err_out, err := ListenOnlyAPRSIS()
	if err != nil {
		log.Fatalf("Failed to connect to server: %s", err)
	}

	ConsolePrinter(tnc2_out, err_out)

}
