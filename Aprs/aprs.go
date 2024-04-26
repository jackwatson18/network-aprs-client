package Aprs

import (
	"crypto/md5"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const (
	MsgPosWithoutTimeNotMessageCapable = iota
	MsgMessage
	MsgPosWithoutTimeMessageCapable
	MsgStatus
)

type APRS_Packet struct {
	Id            int
	Original_AX25 AX25_struct
	Src_callsign  string
	Src_ssid      uint8
	Dst_callsign  string
	Dst_ssid      uint8
	Path          []AX25_Address
	Msg_type      int
	Latitude      float64
	Longitude     float64
	Heading       int
	Speed         int
	Altitude      int
	Comment       string
	SymbolTableId string
	SymbolId      string
	Raw           string
}

var AX25_SSID_BITMASK byte = 0xf

type AX25_Address struct {
	Callsign string
	Ssid     uint8 // AX.25 spec only gives 8 bytes
}

type AX25_struct struct {
	Dst  AX25_Address // dst is not really needed for APRS but we will retain it for compatibility
	Src  AX25_Address
	Path []AX25_Address
	Raw  string
}

func WriteAPRSPacketToDB(packet APRS_Packet) {
	db, err := sql.Open("sqlite3", "./aprs.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	sqlStatement := `
	INSERT INTO aprs (
		send_callsign,
		send_ssid,
		dest_callsign,
		dest_ssid,
		longitude,
		latitude,
		heading,
		speed,
		altitude,
		comment,
		symbolTableId,
		symbolId,
		raw
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
	`

	stmt, err := tx.Prepare(sqlStatement)
	if err != nil {
		log.Fatal("APRS", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		packet.Original_AX25.Src.Callsign,
		packet.Original_AX25.Src.Ssid,
		packet.Original_AX25.Dst.Callsign,
		packet.Original_AX25.Dst.Ssid,
		packet.Longitude,
		packet.Latitude,
		packet.Heading,
		packet.Speed,
		packet.Altitude,
		packet.Comment,
		packet.SymbolTableId,
		packet.SymbolId,
		packet.Original_AX25.Raw)
	if err != nil {
		log.Fatal(err)
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

}

func PrintHexBytes(input []byte) {
	for i := 0; i < len(input); i++ {
		fmt.Printf("%x ", input[i]>>1)
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
	address := AX25_Address{Callsign: call, Ssid: uint8(output[6] & AX25_SSID_BITMASK)}
	// Only part of the last byte refers to the address/SSID. The funky bitmasking above makes sure we only care about the last bits.

	return address
}

func UnparseAX25Address(addr AX25_Address) []byte {
	output := make([]byte, 7)
	for i, char := range addr.Callsign {
		output[i] = byte(char) << 1
	}

	output[6] = addr.Ssid << 1
	fmt.Println(string(output))
	return output
}

// for sending data out
func APRS_to_AX25(packet APRS_Packet) (AX25_struct, error) {
	new_ax25 := AX25_struct{}
	dst := AX25_Address{Callsign: packet.Dst_callsign, Ssid: packet.Dst_ssid}
	src := AX25_Address{Callsign: packet.Src_callsign, Ssid: packet.Src_ssid}
	new_ax25.Src = src
	new_ax25.Dst = dst
	new_ax25.Path = packet.Path

	if packet.Latitude != 0 || packet.Longitude != 0 {
		new_ax25.Raw = encodeCoords(packet)
	}
	if packet.Altitude != 0 {
		new_ax25.Raw = new_ax25.Raw + EncodeAltitude(packet.Altitude)
	}
	new_ax25.Raw = new_ax25.Raw + packet.Comment

	return new_ax25, nil

}

func EncodeAltitude(alt int) string {
	output := "/A=" // prefix
	str_alt := strconv.Itoa(alt)
	for i := len(str_alt); i < 6; i++ {
		output = output + "0"
	}
	output = output + str_alt
	return output
}

func encodeCoords(packet APRS_Packet) string {
	lat := packet.Latitude
	long := packet.Longitude
	// lat first
	output := "="
	var latSymbol string
	if lat >= 0 {
		latSymbol = "N"
	} else {
		latSymbol = "S"
		lat = lat * -1
	}
	lat_deg := math.Floor(lat)
	lat_minutes_unrounded := (lat - lat_deg) * 60
	lat_minutes := math.Floor(lat_minutes_unrounded)
	lat_seconds := math.Floor((lat_minutes_unrounded - lat_minutes) * 100)

	// pad out degrees in lat
	lat_deg_str := strconv.Itoa(int(lat_deg))
	for i := len(lat_deg_str); i < 2; i++ {
		output = output + "0"
	}
	output = output + lat_deg_str

	// pad out minute in lat
	lat_min_str := strconv.Itoa(int(lat_minutes))
	for i := len(lat_min_str); i < 2; i++ {
		output = output + "0"
	}
	output = output + lat_min_str

	// add decimal point
	output = output + "."

	// add seconds and pad as necessary
	lat_sec_str := strconv.Itoa(int(lat_seconds))
	for i := len(lat_sec_str); i < 2; i++ {
		output = output + "0"
	}
	output = output + lat_sec_str

	// add N/S symbol
	output = output + latSymbol
	// add symbol table ID
	output = output + packet.SymbolTableId

	// now do longitude
	var longSymbol string
	if long >= 0 {
		longSymbol = "E"
	} else {
		longSymbol = "W"
		long = long * -1
	}

	long_degrees := math.Floor(long)
	long_mins_unrounded := (long - long_degrees) * 60
	long_mins := math.Floor(long_mins_unrounded)
	long_seconds := math.Floor((long_mins_unrounded - long_mins) * 100)

	// convert to string and add
	long_deg_str := strconv.Itoa(int(long_degrees))
	for i := len(long_deg_str); i < 3; i++ {
		output = output + "0"
	}
	output = output + long_deg_str

	// do minutes
	long_min_str := strconv.Itoa(int(long_mins))
	for i := len(long_min_str); i < 2; i++ {
		output = output + "0"
	}
	output = output + long_min_str
	output = output + "."

	long_seconds_str := strconv.Itoa(int(long_seconds))
	for i := len(long_seconds_str); i < 2; i++ {
		output = output + "0"
	}
	output = output + long_seconds_str
	output = output + longSymbol

	// finally, apend symbol  identifier
	output = output + packet.SymbolId
	return output
}

func ModifiedAX25_to_bytes(frame AX25_struct) []byte {
	// capture an existing valid frame header and use it to package data
	_, raw, _ := ReadAX25FromFile("packetFiles/test2.ax25")
	raw_frame := raw
	for i, b := range raw {
		if b == 0xf0 {
			raw_frame = raw_frame[:i+1]
		}
	}
	raw_frame = append(raw_frame, []byte(frame.Raw)...)

	return raw_frame
}

// end up not using this function, there are undocumented quirks in the spec that made this not compatible with software modem
func AX25_to_bytes(frame AX25_struct) ([]byte, error) {
	bytes := make([]byte, 0)
	// flag byte indicates start of frame or end of frame
	bytes = append(bytes, 0xc0)
	bytes = append(bytes, 0x00) // KISS command byte, will always be zero in our use case
	// Must be 6 bytes. If not 6 bytes, pad with spaces.

	// ax25 flag
	// bytes = append(bytes, 0x7e)

	// repeat padding for Src
	for len(frame.Dst.Callsign) < 6 {
		frame.Dst.Callsign = frame.Dst.Callsign + " "
	}

	raw_dst := UnparseAX25Address(frame.Dst)
	bytes = append(bytes, raw_dst...)

	for len(frame.Src.Callsign) < 6 {
		frame.Src.Callsign = frame.Src.Callsign + " "
	}

	raw_src := UnparseAX25Address(frame.Src)
	bytes = append(bytes, raw_src...)

	// loop over list of addresses in path and do same stuff then add them in.
	for _, path_addr := range frame.Path {
		for len(path_addr.Callsign) < 6 {
			path_addr.Callsign = path_addr.Callsign + " "
		}
		raw_path := UnparseAX25Address(path_addr)
		bytes = append(bytes, raw_path...)
	}

	// UI control field and protocol field bytes (Per APRS documentation)
	bytes = append(bytes, 0x03)
	bytes = append(bytes, 0xf0)

	bytes = append(bytes, []byte(frame.Raw)...)

	// at this point a checksum should be generated in order to meet spec, but the means of doing so are nontrivial. left blank for now.
	// bytes = append(bytes, 0xf0)
	// bytes = append(bytes, 0x0f)

	// ax25 flag
	// bytes = append(bytes, 0x7e)
	// flag byte indicates start of frame or end of frame
	bytes = append(bytes, 0xc0)

	return bytes, nil
}

func ParseAX25(raw_frame []byte) AX25_struct {
	// WriteBytesToFile(raw_frame, "packetFiles/test2.ax25")
	PrintHexBytes(raw_frame)
	frame := raw_frame[2 : len(raw_frame)-1] // strip control headers
	//PrintHexBytes(frame)
	ax := AX25_struct{}
	// left shift the src/dst bytes according to AX.25 standard
	dst := ParseAX25Address(frame[0:7])
	src := ParseAX25Address(frame[7:14])
	// fmt.Println(src)
	// fmt.Println(dst)
	ax.Dst = dst
	ax.Src = src
	frame = frame[14:]
	for len(frame) > 7 && frame[0] != 3 {
		ax.Path = append(ax.Path, ParseAX25Address(frame[:7]))
		frame = frame[7:]
	}
	ax.Raw = string(frame[2:])
	//ax.raw = strings.Replace(ax.raw, "\r", "", -1)
	return ax

}

func DisplayAX25Packet(p AX25_struct) {
	pathstr := ""
	for i, path_id := range p.Path {
		tmpstr := fmt.Sprintf("%s-%d", path_id.Callsign, path_id.Ssid)
		if i > 0 {
			pathstr = pathstr + " "
		}
		pathstr = pathstr + tmpstr

	}
	fmt.Printf("Dst ID: %s-%d\n", p.Dst.Callsign, p.Dst.Ssid)
	fmt.Printf("Src ID: %s-%d\n", p.Src.Callsign, p.Src.Ssid)
	fmt.Printf("Path  : %s\n", pathstr)
	fmt.Printf("Msg   : %s\n", p.Raw)
	fmt.Println()

}

func parseAPRS(p AX25_struct) (APRS_Packet, error) {
	msgType := -1
	// fmt.Println()
	packet := APRS_Packet{Original_AX25: p}
	packet.Raw = p.Raw
	packet.Src_callsign = p.Src.Callsign
	packet.Src_ssid = p.Src.Ssid
	packet.Dst_callsign = p.Dst.Callsign
	packet.Dst_ssid = p.Dst.Ssid

	switch p.Raw[0] {
	case '!':
		msgType = MsgPosWithoutTimeNotMessageCapable
	case '=':
		msgType = MsgPosWithoutTimeMessageCapable
	case '>':
		msgType = MsgStatus
	case ':':
		msgType = MsgMessage
	}
	if msgType == -1 {
		return packet, fmt.Errorf("parseAPRStype: Invalid packet type '%s'", string(p.Raw[0]))
	}

	packet.Msg_type = msgType
	var err error = nil
	switch packet.Msg_type {
	case MsgPosWithoutTimeNotMessageCapable:
		// fmt.Println("ParsingPosition...")
		packet, err = parseAPRSPositionNoTime(packet)

	case MsgPosWithoutTimeMessageCapable:
		// fmt.Println("ParsingPosition...")
		packet, err = parseAPRSPositionNoTime(packet)
	case MsgMessage:
		packet.Comment = p.Raw
	case MsgStatus:
		packet, err = parseAPRSPositionNoTime(packet)
	}

	if err != nil {
		fmt.Printf("ERR: %s\n", err)
	}

	// fmt.Printf("PACKET TO FOLLOW: ")
	fmt.Println()
	fmt.Println(packet)
	return packet, err

}

func extractExtentionData(data string, packet APRS_Packet) (string, APRS_Packet, error) {
	if len(data) < 7 {
		return data, packet, fmt.Errorf("extractExtentionData: data length must be 7 bytes or more but got %d", len(data))
	}

	if data[3] == '/' { // CSE/SPD
		heading, err := strconv.Atoi(data[0:3])
		if err != nil {
			return data, packet, err
		}
		speed, err := strconv.Atoi(data[4:7])
		if err != nil {
			return data, packet, err
		}

		packet.Heading = heading
		packet.Speed = speed
		data = data[7:]

	}

	return data, packet, nil
}

func extractAltitudeFromCommentText(data string, packet APRS_Packet) (string, APRS_Packet, error) {
	if data[0:3] == "/A=" {
		if len(data) < 9 {
			return data, packet, errors.New("extractAltitude: detected altitude but length of data to short")
		}
		altitude, err := strconv.Atoi(data[3:9])
		if err != nil {
			return data, packet, err
		}

		packet.Altitude = altitude
		data = data[9:]
	}
	return data, packet, nil
}

func parseAPRSPositionNoTime(packet APRS_Packet) (APRS_Packet, error) {
	packetText := packet.Original_AX25.Raw[1:] // strip off identifer byte
	// fmt.Printf("****PacketText: %s\n", packetText)
	if len(packetText) < 18 {
		return packet, errors.New("parseAPRSPositionNoTime: Packet text is too short")
	}
	raw_lat := packetText[0:8]
	symbolTableID := packetText[8]
	raw_long := packetText[9:18]
	symbolCode := packetText[18]
	// extentionData := packetText[19:26]
	// fmt.Println(extentionData)
	// fmt.Println(packetText[26:])

	latitude, longitude, err := AnalogToDigitalAPRSCoords(raw_lat, raw_long)
	if err != nil {
		return packet, err
	}
	// after this point, data existing is not guaranteed. Use consumer model.
	packetText, packet, err = extractExtentionData(packetText[19:], packet)
	if err != nil {
		return packet, err
	}
	packetText, packet, err = extractAltitudeFromCommentText(packetText, packet)
	if err != nil {
		return packet, err
	}
	// altitude, comment, err := checkAndPullAltitudeFromComment(packetText[19:])
	// if err != nil {
	// 	log.Default().Println("Error encountered parsing altitude from comment.")
	// }

	comment := strings.Trim(packetText, " ") // remove any extra spaces

	packet.Latitude = latitude
	packet.Longitude = longitude

	packet.SymbolTableId = string(symbolTableID)
	packet.SymbolId = string(symbolCode)
	packet.Comment = comment

	//fmt.Printf("lat: %f, long: %f symbol:%s%s, heading: %d, speed: %dkts, altitude: %dft, comment: %s\n", latitude, longitude, string(symbolTableID), string(symbolCode), heading, speed, altitude, comment)
	// fmt.Println(packet)
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

func SendToModem(server string, data []byte) error {
	conn, err := net.Dial("tcp", server)
	if err != nil {
		log.Fatalf("sendToModem: Error connecting to modem: %s", err)
	}
	defer conn.Close()

	message := make([]byte, 0)
	message = append(message, byte(0xC0))
	message = append(message, data...)
	message = append(message, byte(0xC0))

	conn.Write(message)
	return nil
}

func ConnectionLoop(server string) {
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
		// filepath := fmt.Sprintf("packetFiles/%x.ax25", h.Sum(nil))
		// WriteBytesToFile(buffer[:mLen], filepath)
		ax_25 := ParseAX25(buffer[:mLen])

		// write bytes to file for testing

		// DisplayAX25Packet(ax_25)
		packet, err := parseAPRS(ax_25)
		if err != nil {
			fmt.Printf("error parsing packet: %s\n \t raw packet: %s", err, packet.Raw)
		} else {
			WriteAPRSPacketToDB(packet)
		}

	}
}

func WriteBytesToFile(data []byte, filename string) {
	err := os.WriteFile(filename, data, 0775)
	if err != nil {
		log.Fatal(err)
	}
}

func ReadAX25FromFile(filename string) (AX25_struct, []byte, error) {
	data, err := os.ReadFile(filename)

	if err != nil {
		log.Fatal(err)
	}

	result := ParseAX25(data)
	DisplayAX25Packet(result)

	return result, data, nil
}

func TestConversions(server string) {
	result, raw, _ := ReadAX25FromFile("packetFiles/test2.ax25")
	SendToModem(server, raw)
	fmt.Println("BYTES IN:")
	PrintHexBytes(raw)
	aprs_packet, _ := parseAPRS(result)
	converted_ax, _ := APRS_to_AX25(aprs_packet)
	raw_bytes := ModifiedAX25_to_bytes(converted_ax)
	fmt.Println("BYTES OUT:")
	PrintHexBytes(raw_bytes)
	SendToModem(server, raw_bytes)

}

func TestCallsignShifting() {
	addr := AX25_Address{Callsign: "KK7EWJ", Ssid: 0}
	raw_addr := UnparseAX25Address(addr)
	unraw_addr := ParseAX25Address(raw_addr)
	fmt.Println(unraw_addr)
}

func TestEncodeAlt() {
	fmt.Println(EncodeAltitude(123))
}

func TestEncodeCoords() {
	packet := APRS_Packet{Latitude: 34.15, Longitude: -117.500, SymbolTableId: "/", SymbolId: "-"}
	encodedCoords := encodeCoords(packet)
	fmt.Println(encodedCoords)
}

func TestEncodeAndSend(server string) {
	packet := APRS_Packet{
		Latitude:      46.21235,
		Longitude:     -117.1235,
		SymbolTableId: "/",
		SymbolId:      "-",
		Altitude:      2134,
		Comment:       "Test encode",
	}
	ax25, _ := APRS_to_AX25(packet)
	raw := ModifiedAX25_to_bytes(ax25)
	SendToModem(server, raw)
}
