package main

import (
	"fmt"
	"net"
)

func PrintHexBytes(input []byte) {
	for i := 0; i < len(input); i++ {
		fmt.Printf("%c ", input[i])
	}
	fmt.Print("\n")
	fmt.Println(string(input))
}

func ParseAX25(raw_packet []byte) {
	packet := raw_packet[2 : len(raw_packet)-1]
	// strip the "end frame" chars from packet
	dst_callsign := packet[0:7]
	src_callsign := packet[7:15]
	for i := 0; i < 7; i++ {
		dst_callsign[i] = dst_callsign[i] >> 1
		src_callsign[i] = src_callsign[i] >> 1
	}
	PrintHexBytes(packet)

}

func main() {
	conn, err := net.Dial("tcp", "localhost:8001")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	buffer := make([]byte, 1024)
	mLen, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	}
	fmt.Println("Recieved:")
	PrintHexBytes(buffer[:mLen])
	fmt.Println("Parsed:")
	ParseAX25(buffer[:mLen])
}
