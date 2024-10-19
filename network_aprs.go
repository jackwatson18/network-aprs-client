package main

import (
	"flag"
	"fmt"
	"internal/AX25"
	"log"
	"net"

	"github.com/fatih/color"
	_ "github.com/mattn/go-sqlite3"
)

func ListenOnlyLoop(server string) {
	conn, err := net.Dial("tcp", server)

	if err != nil {
		log.Fatalf("Error connecting to modem: %s", err)
	}

	defer conn.Close()
	fmt.Printf("Connected to KISS Server at %s\n", server)

	buffer := make([]byte, 1024)

	for {
		mLen, err := conn.Read(buffer)

		if err != nil {
			log.Fatalf("Error reading conn: %s", err)
		}

		frame_struct, err := AX25.ConvertBytesToAX25(buffer[:mLen])
		if err != nil {
			fmt.Printf("%v\n", err)
			continue
		}
		c := color.New(color.FgGreen).Add(color.Bold)
		c.Println("New Packet:")
		fmt.Printf("%v\n", frame_struct.String())
	}
}

func main() {

	serverPtr := flag.String("srv", "localhost:8001", "KISS Server Address")
	flag.Parse()
	fmt.Println(*serverPtr)

	ListenOnlyLoop(*serverPtr)

}
