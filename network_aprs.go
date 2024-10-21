package main

import (
	"flag"
	"fmt"
	"internal/AX25"
	"log"
	"net"

	"github.com/fatih/color"
)

// connects to, and listens to a KISS server. Returns a channel. Meant to run as a goroutine.
func KISSServerConnector(server string, ax25_chan chan AX25.AX25_frame, error_chan chan error) {
	conn, err := net.Dial("tcp", server)

	if err != nil {
		error_chan <- fmt.Errorf("KISSServerConnector: error connecting to modem: %v", err)
		return
	}

	defer conn.Close()
	fmt.Printf("KISSServerConnector connected to KISS Server at %s\n", server)

	buffer := make([]byte, 1024)

	for {
		mLen, err := conn.Read(buffer)

		if err != nil {
			error_chan <- fmt.Errorf("KISSServerConnector: error reading conn: %v", err)
			return
		}

		trimmed_bytes, err := AX25.StripKISSWrapper(buffer[:mLen])
		if err != nil {
			fmt.Printf("%v\n", err)
			continue
		}
		frame_struct, err := AX25.ConvertBytesToAX25(trimmed_bytes)
		if err != nil {
			fmt.Printf("%v\n", err)
			continue
		}

		ax25_chan <- frame_struct
	}
}

func ListenOnlyLoop(frame_chan chan AX25.AX25_frame, error_chan chan error) error {
	c := color.New(color.FgGreen).Add(color.Bold)

	for {
		select {
		case err := <-error_chan:
			return err
		case ax25_frame := <-frame_chan:
			c.Println("New Packet:")
			fmt.Printf("%v\n", ax25_frame.TNC2())

		}
	}
}

func main() {

	serverPtr := flag.String("srv", "localhost:8001", "KISS Server Address")
	flag.Parse()
	fmt.Println(*serverPtr)

	frame_chan := make(chan AX25.AX25_frame)
	error_chan := make(chan error)

	go KISSServerConnector(*serverPtr, frame_chan, error_chan)

	err := ListenOnlyLoop(frame_chan, error_chan)
	if err != nil {
		log.Fatalf("%v", err)
	}

}
