package main

import (
	"fmt"
	"net"
	"strings"
)

type APRS_Address struct {
	callsign string
	Ssid     uint8 // AX.25 spec only gives 8 bytes
}

type AX25_struct struct {
	dst  APRS_Address // dst is not really needed for APRS but we will retain it for compatibility
	src  APRS_Address
	path []APRS_Address
	raw  string
}

func PrintHexBytes(input []byte) {
	for i := 0; i < len(input); i++ {
		fmt.Printf("%c ", input[i]>>1)
	}
	fmt.Print("\n")
	// fmt.Println(string(input))
}

func ParseAX25Address(raw []byte) APRS_Address {
	output := make([]byte, len(raw))
	for i, char := range raw {
		output[i] = char >> 1
	}

	call := strings.TrimSpace(string(output[0:6]))
	address := APRS_Address{callsign: call, Ssid: uint8(output[6] & 0xf)}

	return address
}

func ParseAX25(raw_frame []byte) {
	frame := raw_frame[:len(raw_frame)-1] // strip control headers
	PrintHexBytes(frame)
	ax := AX25_struct{}
	// left shift the src/dst bytes according to AX.25 standard
	dst := ParseAX25Address(frame[1:8])
	src := ParseAX25Address(frame[9:16])
	// fmt.Println(src)
	// fmt.Println(dst)
	ax.dst = dst
	ax.src = src
	frame = frame[16:]
	for len(frame) > 7 && frame[0] != 3 {
		ax.path = append(ax.path, ParseAX25Address(frame[:7]))
		frame = frame[7:]
	}
	ax.raw = string(frame[2:])
	fmt.Println(ax)

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
