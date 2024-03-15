package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	MsgPosWithoutTimeNotMessageCapable = iota
	MsgPositionWithTimestamp
	MsgMessage
	MsgObject
	MsgPosWithoutTimeMessageCapable
	MsgQuery
	MsgTelemetry
	MsgMicE
)

type APRS_Packet struct {
	original_AX25 AX25_struct
	msg_type      int
}

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

func parseAPRS(p AX25_struct) (APRS_Packet, error) {
	msgType := -1
	switch p.raw[0] {
	case '!':
		msgType = MsgPosWithoutTimeNotMessageCapable
	case '=':
		msgType = MsgPosWithoutTimeMessageCapable
	}
	if msgType == -1 {
		return APRS_Packet{}, errors.New("parseAPRStype: Invalid packet type")
	}

	packet := APRS_Packet{original_AX25: p, msg_type: msgType}
	switch packet.msg_type {
	case MsgPosWithoutTimeNotMessageCapable:
		packet, _ = parseAPRSPositionNoTime(packet)
	case MsgPosWithoutTimeMessageCapable:
		packet, _ = parseAPRSPositionNoTime(packet)
	}

	return packet, nil

}

func parseAPRSPositionNoTime(packet APRS_Packet) (APRS_Packet, error) {
	packetText := packet.original_AX25.raw[1:] // strip off identifer byte
	raw_lat := packetText[0:8]
	symbolTableID := packetText[8]
	raw_long := packetText[9:18]
	symbolCode := packetText[18]
	comment := packetText[19:]

	latitude, longitude, _ := AnalogToDigitalAPRSCoords(raw_lat, raw_long)

	fmt.Printf("lat: %f, long: %f symbol:%s%s, comment: %s", latitude, longitude, string(symbolTableID), string(symbolCode), comment)
	return packet, nil

}

func AnalogToDigitalAPRSCoords(alat string, along string) (float64, float64, error) {
	// Takes coordinate data as byte array, returns as coordinate pair.
	// APRS gives us coordinates as old school hours, minutes...we want a pure integer coordinate.
	if len(alat) != 8 {
		return 0, 0, errors.New("AnalogToDigitalAPRSCoords: Invalid analog latitude length")
	}
	if len(along) != 9 {
		return 0, 0, errors.New("AnalogToDigitalAPRSCoords: Invalid analog longitude length")
	}
	lat, long := 0.0, 0.0

	degrees, err := strconv.Atoi(alat[0:2])
	if err != nil {
		return 0, 0, err
	}

	minutes, err := strconv.Atoi(alat[2:4])
	if err != nil {
		return 0, 0, err
	}
	seconds, err := strconv.Atoi(alat[5:7])
	if err != nil {
		return 0, 0, err
	}

	lat = float64(degrees) + float64(minutes)/60 + float64(seconds)/3600
	if alat[7] == 'S' {
		lat = lat * -1
	}

	degrees, err = strconv.Atoi(along[0:3])
	if err != nil {
		return 0, 0, err
	}

	minutes, err = strconv.Atoi(along[3:5])
	if err != nil {
		return 0, 0, err
	}

	seconds, err = strconv.Atoi(along[6:8])
	if err != nil {
		return 0, 0, err
	}

	long = float64(degrees) + float64(minutes)/60 + float64(seconds)/3600
	if along[8] == 'W' {
		long = long * -1
	}

	return lat, long, nil
}

func MonitorLoop(server string) {
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
		h := md5.New()
		h.Write(buffer[:mLen])
		filepath := fmt.Sprintf("packetFiles/%x.ax25", h.Sum(nil))
		WriteBytesToFile(buffer[:mLen], filepath)
		ax_25 := ParseAX25(buffer[:mLen])

		// write bytes to file for testing

		DisplayAX25Packet(ax_25)
		parseAPRS(ax_25)
	}
}

func WriteBytesToFile(data []byte, filename string) {
	err := os.WriteFile(filename, data, 0775)
	if err != nil {
		log.Fatal(err)
	}
}

func ReadAX25FromFile(filename string) (AX25_struct, error) {
	data, err := os.ReadFile(filename)

	if err != nil {
		log.Fatal(err)
	}

	result := ParseAX25(data)
	DisplayAX25Packet(result)

	return result, nil
}

func main() {
	server := "localhost:8001"
	if len(os.Args) > 1 {
		server = os.Args[1]
	}
	result, _ := ReadAX25FromFile("packetFiles/820503b7fd4e7f84bfea19411d1abdd0a5d438ca504d2ee71489c80e69ce9446.ax25")
	parseAPRS(result)
	//	fmt.Println(packet)
	MonitorLoop(server)

}
