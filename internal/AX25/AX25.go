// provides the means for decoding, and eventually encoding AX25 data from a TCP connection.
package AX25

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

const FLAG_CHAR byte = 0x7e
const FEND byte = 0xc0
const CONTROL_FIELD byte = 3
const PROTOCOL_ID byte = 0xf0
const AX25_SSID_BITMASK byte = 0xf
const CMD_OR_RPT_BITMASK byte = 0b10000000
const AX25_RESERVED_BITS byte = 0b01100000
const AX25_EXTENTION_BITMASK byte = 0b00000001

type Callsign struct {
	Call       string
	Ssid       uint8
	IsCmdOrRpt bool
}

func (callsign Callsign) String() string {
	if callsign.Ssid == 0 {
		return callsign.Call
	} else {
		return fmt.Sprintf("%s-%v", callsign.Call, callsign.Ssid)
	}
}

// converts callsign to bytes including appropriate bitshifting and SSID bits
func (callsign Callsign) AX25Encode(ssid_bitmask byte) []byte {
	output := make([]byte, 7)
	for i := 0; i < len(output); i++ {
		output[i] = ' ' << 1
	}
	for i, c := range callsign.Call {
		output[i] = byte(c) << 1
	}

	output[6] = (byte(callsign.Ssid) << 1) | ssid_bitmask
	return output
}

type AX25_frame struct {
	Dest_addr   Callsign
	Source_addr Callsign
	Digi_path   []Callsign
	Info_field  string
}

func (frame AX25_frame) StructPrint() string {
	// multiline strings are stupid, ignore the weird formatting.
	return fmt.Sprintf(
		`      Dest: %s
    Source: %s
  DigiPath: %v
Info Field: %s
`, frame.Dest_addr, frame.Source_addr, frame.Digi_path, frame.Info_field)
}

// for Go built-ins that use GoStringer interface. Wraps TNC2 string method
func (frame AX25_frame) String() string {
	return frame.TNC2()
}

// Converts AX25 Frame to TNC2 format string output
func (frame AX25_frame) TNC2() string {
	output := fmt.Sprintf("%s>%s", frame.Source_addr, frame.Dest_addr)

	// has an address in the digipath repeated this frame?
	wasRpt := false
	for _, digi := range frame.Digi_path {
		// if next digi callsign is not repeated, and we've seen repeated, drop it.
		if wasRpt && !digi.IsCmdOrRpt {
			break
		}

		output = output + "," + digi.String()

		if digi.IsCmdOrRpt {
			wasRpt = true
		}
	}
	if wasRpt {
		// if we had a digipeater digipeat the frame, mark last visited digi with a star/astrisk
		output = output + "*"
	}

	output = output + ":" + frame.Info_field

	return output
}

// given an AX25_frame, returns a slice of bytes ready for transmission
func (frame AX25_frame) Bytes(isResponse bool) []byte {
	// TODO: Handle Command/Response part of AX25 standard...
	output := &bytes.Buffer{}
	// write dest callsign
	mask := AX25_RESERVED_BITS
	if !isResponse {
		// if not a response set the CMD bit
		mask = mask | CMD_OR_RPT_BITMASK
	}
	output.Write(frame.Dest_addr.AX25Encode(mask))

	// write the source callsign
	mask = AX25_RESERVED_BITS
	// if there are no digi paths, we need to set the extention bit to 1 indicating its the last callsign in the address field
	if len(frame.Digi_path) == 0 {
		mask = mask | AX25_EXTENTION_BITMASK
	}
	if isResponse {
		mask = mask | CMD_OR_RPT_BITMASK
	}
	output.Write(frame.Source_addr.AX25Encode(mask))

	// write the digipeaters
	for i, digi := range frame.Digi_path {
		mask = AX25_RESERVED_BITS
		if i == len(frame.Digi_path)-1 {
			// last digipeater
			mask = mask | AX25_EXTENTION_BITMASK
		}
		output.Write(digi.AX25Encode(mask))
	}

	// write control/protocol bytes
	output.Write([]byte{CONTROL_FIELD, PROTOCOL_ID})
	// info field
	output.Write([]byte(frame.Info_field))

	return output.Bytes()
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
	cmd_or_rpt := false
	if (addr[6] & CMD_OR_RPT_BITMASK) == CMD_OR_RPT_BITMASK {
		cmd_or_rpt = true
	}

	return Callsign{Call: call, Ssid: ssid, IsCmdOrRpt: cmd_or_rpt}, nil
}
