package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

var AX25_SSID_BITMASK byte = 0xf

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
	address := AX25_Address{callsign: call, Ssid: uint8(output[6] & AX25_SSID_BITMASK)}
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
	//ax.raw = strings.Replace(ax.raw, "\r", "", -1)
	return ax

}

func DisplayAX25Packet(p AX25_struct) {
	pathstr := ""
	for i, path_id := range p.path {
		tmpstr := fmt.Sprintf("%s-%d", path_id.callsign, path_id.Ssid)
		if i > 0 {
			pathstr = pathstr + " "
		}
		pathstr = pathstr + tmpstr

	}
	fmt.Printf("Dst ID: %s-%d\n", p.dst.callsign, p.dst.Ssid)
	fmt.Printf("Src ID: %s-%d\n", p.src.callsign, p.src.Ssid)
	fmt.Printf("Path  : %s\n", pathstr)
	fmt.Printf("Msg   : %s\n", p.raw)
	fmt.Println()

}

func main() {
	server := "localhost:8001"
	if len(os.Args) > 1 {
		server = os.Args[1]
	}

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
		DisplayAX25Packet(ParseAX25(buffer[:mLen]))
	}

}
