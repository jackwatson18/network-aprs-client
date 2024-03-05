package main

import (
	"fmt"
	"net"
	"strings"
)

type AX25_Address struct {
	callsign string
	Ssid     uint8 // AX.25 spec only gives 8 bytes
}

type AX25_struct struct {
	dst  AX25_Address // dst is not really needed for APRS but we will retain it for compatibility
	src  AX25_Address
	path []AX25_Address
	raw  string
}

func PrintHexBytes(input []byte) {
	for i := 0; i < len(input); i++ {
		fmt.Printf("%c", input[i]>>1)
	}
	fmt.Print("\n")
	// fmt.Println(string(input))
}

func ParseAX25Address(raw []byte) AX25_Address {
	output := make([]byte, len(raw))
	for i, char := range raw {
		output[i] = char >> 1
	}

	call := strings.TrimSpace(string(output[0:6]))
	address := AX25_Address{callsign: call, Ssid: uint8(output[6] & 0xf)}
	// Only part of the last byte refers to the address/SSID. The funky bitmasking above makes sure we only care about the last bits.

	return address
}

func ParseAX25(raw_frame []byte) AX25_struct {
	frame := raw_frame[2 : len(raw_frame)-1] // strip control headers
	//PrintHexBytes(frame)
	ax := AX25_struct{}
	// left shift the src/dst bytes according to AX.25 standard
	dst := ParseAX25Address(frame[0:7])
	src := ParseAX25Address(frame[7:14])
	// fmt.Println(src)
	// fmt.Println(dst)
	ax.dst = dst
	ax.src = src
	frame = frame[14:]
	for len(frame) > 7 && frame[0] != 3 {
		ax.path = append(ax.path, ParseAX25Address(frame[:7]))
		frame = frame[7:]
	}
	ax.raw = string(frame[2:])
	return ax

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
	fmt.Println(ParseAX25(buffer[:mLen]))
}
