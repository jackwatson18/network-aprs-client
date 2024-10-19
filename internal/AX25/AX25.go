package AX25

import (
	"fmt"
	"os"
	"strings"
)

const FLAG_CHAR byte = 0x7e
const FEND byte = 0xc0
const CONTROL_FIELD byte = 3
const PROTOCOL_ID byte = 0xf0
const AX25_SSID_BITMASK byte = 0xf

type Callsign struct {
	Call string
	Ssid uint8
}

func (callsign Callsign) String() string {
	if callsign.Ssid == 0 {
		return callsign.Call
	} else {
		return fmt.Sprintf("%s-%v", callsign.Call, callsign.Ssid)
	}
}

func (callsign Callsign) GoString() string {
	return callsign.String()
}

type AX25_frame struct {
	Dest_addr   Callsign
	Source_addr Callsign
	Digi_path   []Callsign
	Info_field  string
}

func (frame AX25_frame) String() string {
	// multiline strings are stupid, ignore the weird formatting.
	return fmt.Sprintf(
		`      Dest: %s
    Source: %s
  DigiPath: %v
Info Field: %s
`, frame.Dest_addr, frame.Source_addr, frame.Digi_path, frame.Info_field)
}

func (frame AX25_frame) GoString() string {
	return frame.String()
}

// Converts AX25 Frame to TNC2 format string output
func (frame AX25_frame) TNC2() string {
	output := fmt.Sprintf("%s>%s", frame.Source_addr, frame.Dest_addr)

	for _, digi := range frame.Digi_path {
		output = output + "," + digi.String()
	}

	return output
}

func readBytesFromFile(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)

	return data, err
}

// strips off FEND bytes that KISS frame wraps around data.
func StripKISSWrapper(data []byte) ([]byte, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf(`stripKISSWrapper: Data would be empty after stripping FEND and command bytes`)
	}
	if data[0] != FEND || data[len(data)-1] != FEND {
		return nil, fmt.Errorf(`stripKISSWrapper: Frame start and end should be %x, but got %x and %x`, FEND, data[0], data[len(data)-1])
	}

	return data[2 : len(data)-1], nil
}

// Converts raw byte AX.25 frame into an AX25_frame struct.
func ConvertBytesToAX25(data []byte) (AX25_frame, error) {
	// data length must be in range [21, 332] inclusive.
	if len(data) < 19 || len(data) > 330 {
		return AX25_frame{}, fmt.Errorf(`ConvertBytesToAX25: number of bytes must be between 21 and 332 but got %d`, len(data))
	}

	dest_addr, err := parseAddr(data[:7])
	if err != nil {
		return AX25_frame{}, fmt.Errorf(`ConvertBytesToAX25: %w`, err)
	}
	data = data[7:]

	source_addr, err := parseAddr(data[:7])
	if err != nil {
		return AX25_frame{}, fmt.Errorf(`ConvertBytesToAX25: %w`, err)
	}
	data = data[7:]

	// parse digipeater path
	var digi_path []Callsign
	for len(data) > 7 && data[0] != CONTROL_FIELD { // Control field separating Digi path and rest of frame always 3
		digi_addr, err := parseAddr(data[:7])
		if err != nil {
			return AX25_frame{}, fmt.Errorf(`ConvertBytesToAX25: %w`, err)
		}

		digi_path = append(digi_path, digi_addr)
		data = data[7:]
	}

	// if data isn't long enough to have CONTROL and PROTOCOL fields, or they're just not there, invalid AX.25 frame
	if len(data) < 2 || data[0] != CONTROL_FIELD || data[1] != PROTOCOL_ID {
		return AX25_frame{}, fmt.Errorf("ConvertBytesToAX25: Frame missing CONTROL_FIELD and PROTOCOL_ID bytes")
	}

	// information field
	info_field := string(data[2:])

	return AX25_frame{Dest_addr: dest_addr, Source_addr: source_addr, Digi_path: digi_path, Info_field: info_field}, nil

}

// converts Destination or Source address in AX.25 frame to a string readable format.
func parseAddr(addr []byte) (Callsign, error) {
	// addr is ALWAYS 7 bytes.
	if len(addr) != 7 {
		return Callsign{}, fmt.Errorf(`bytesToAddr: addr must be 7 bytes but got %v`, len(addr))
	}

	output := make([]byte, len(addr))
	for i, char := range addr {
		output[i] = char >> 1
	}

	call := strings.TrimSpace(string(output[0:6]))
	ssid := uint8(output[6] & AX25_SSID_BITMASK)

	return Callsign{Call: call, Ssid: ssid}, nil
}
