package AX25

import (
	"bytes"
	"fmt"
	"testing"
)

func Test_readBytesFromFile_validFile(t *testing.T) {
	_, err := readBytesFromFile("./test.ax25")

	if err != nil {
		t.Fatalf(`%v`, err)
	}

}

func Test_stripKISSWrapper_FailIfTooShort(t *testing.T) {
	data := make([]byte, 3)
	_, err := StripKISSWrapper(data)

	if err == nil {
		t.Errorf("Expected an error about input too short, but stripKISSWrapper generated no error")
	}
}

func Test_stripKISSWrapper_FailFirstNotFEND(t *testing.T) {
	data := make([]byte, 4)
	_, err := StripKISSWrapper(data)

	if err == nil {
		t.Errorf("Expected an error about no FEND wrapper, but stripKISSWrapper generated no error")
	}
}

func Test_stripKISSWrapper_FailLastNotFEND(t *testing.T) {
	data := make([]byte, 4)
	data[0] = FEND
	_, err := StripKISSWrapper(data)

	if err == nil {
		t.Errorf("Expected an error about no FEND wrapper, but stripKISSWrapper generated no error")
	}
}

func Test_stripKISSWrapper_Succeed(t *testing.T) {
	data := make([]byte, 4)
	data[0] = FEND
	data[3] = FEND
	_, err := StripKISSWrapper(data)

	if err != nil {
		t.Fatalf(`%v`, err)
	}
}

func Test_ReadBytesFromFile_NonexistantFile(t *testing.T) {
	_, err := readBytesFromFile("./badDirectory/badFile.ax25")

	if err == nil {
		t.Errorf("Expected an error, but ReadBytesFromFile did not produce error")
	}

}

func Test_ConvertBytesToAX25_TooShort(t *testing.T) {
	_, err := ConvertBytesToAX25([]byte("foo"))
	if err == nil {
		t.Errorf("Expected an error about input too short, but CovertBytesToAX25 generated no error")
	}

}

func Test_ConvertBytesToAX25_TooLong(t *testing.T) {
	_, err := ConvertBytesToAX25(make([]byte, 333))
	if err == nil {
		t.Errorf("Expected an error about input too short, but CovertBytesToAX25 generated no error")
	}

}

func Test_ConvertBytesToAX25_RunsWithoutErrorOnTestFile(t *testing.T) {
	raw, err := readBytesFromFile("./test.ax25")
	if err != nil {
		t.Fatalf(`%v`, err)
	}

	ax25_raw, err := StripKISSWrapper(raw)
	if err != nil {
		t.Fatalf(`%v`, err)
	}

	output, err := ConvertBytesToAX25(ax25_raw)
	if err != nil {
		t.Fatalf(`%v`, err)
	}
	fmt.Printf("%v\n", output)

}

func Test_parseAddr_FailIfTooShort(t *testing.T) {
	_, err := parseAddr([]byte("SHORT"))

	if err == nil {
		t.Fatalf("Expected an error about length but not nothing")
	}
}

func Test_parseAddr_FailIfTooLong(t *testing.T) {
	_, err := parseAddr([]byte("VERYLONG"))

	if err == nil {
		t.Fatalf("Expected an error about length but not nothing")
	}
}

func Test_parseAddr_CmdBitSet(t *testing.T) {
	result, err := parseAddr([]byte{0x96, 0x96, 0x6E, 0x8A, 0xAE, 0x94, 0xEF})
	expect := Callsign{Call: "KK7EWJ", Ssid: 7, IsCmdOrRpt: true}
	if err != nil {
		t.Errorf("Expected no error but got %v", err)
	}

	if result.IsCmdOrRpt != expect.IsCmdOrRpt {
		t.Errorf("Expected %v with CMD=%v but got %v with CMD=%v", expect, expect.IsCmdOrRpt, result, result.IsCmdOrRpt)

	}
}

func Test_parseAddr_CorrectOutput(t *testing.T) {
	result, err := parseAddr([]byte{0x96, 0x96, 0x6E, 0x8A, 0xAE, 0x94, 0x6F})
	expect := Callsign{Call: "KK7EWJ", Ssid: 7}

	if err != nil {
		t.Errorf("Expected no error but got %v", err)
	}

	if result != expect {
		t.Errorf("Expected %v but got %v", expect, result)
	}
}

func Test_AX25_frame_StructPrint(t *testing.T) {
	dst := Callsign{Call: "KK7EWJ", Ssid: 0}
	src := Callsign{Call: "N0CALL", Ssid: 0}
	path := []Callsign{{Call: "WIDE1", Ssid: 1}, {Call: "WIDE2", Ssid: 1}}
	input := AX25_frame{Dest_addr: dst, Source_addr: src, Digi_path: path, Info_field: "foobar"}.StructPrint()
	expect := `      Dest: KK7EWJ
    Source: N0CALL
  DigiPath: [WIDE1-1 WIDE2-1]
Info Field: foobar
`

	if input != expect {
		t.Errorf(`Input and expected do not match.
	Expected:
	%v
	Actual:
	%v`, expect, input)
	}
}

func Test_AX25_frame_TNC2_String(t *testing.T) {
	src := Callsign{Call: "KK7EWJ", Ssid: 7}
	dst := Callsign{Call: "N0CALL", Ssid: 2}
	digi := []Callsign{{Call: "WIDE2", Ssid: 1}, {Call: "RS0ISS", IsCmdOrRpt: true}}
	result := AX25_frame{Source_addr: src, Dest_addr: dst, Digi_path: digi, Info_field: "foobartest"}.String()
	expected := "KK7EWJ-7>N0CALL-2,WIDE2-1,RS0ISS*:foobartest"
	if result != expected {
		t.Errorf("Expected %s but got %s", expected, result)
	}
}

func Test_AX25_frame_TNC2_ignores_digi_path_after_repeated_call(t *testing.T) {
	src := Callsign{Call: "KK7EWJ", Ssid: 7}
	dst := Callsign{Call: "N0CALL", Ssid: 2}
	result := AX25_frame{Source_addr: src, Dest_addr: dst, Digi_path: []Callsign{{Call: "RS0ISS", IsCmdOrRpt: true}, {Call: "WIDE2", Ssid: 1}}}.TNC2()
	expected := "KK7EWJ-7>N0CALL-2,RS0ISS*:"
	if result != expected {
		t.Errorf("Expected %s but got %s", expected, result)
	}
}

func Test_AX25_frame_TNC2_multiple_repeaters_only_star_last_repeater(t *testing.T) {
	src := Callsign{Call: "KK7EWJ", Ssid: 7}
	dst := Callsign{Call: "N0CALL", Ssid: 2}
	digi := []Callsign{{Call: "WIDE1", IsCmdOrRpt: true}, {Call: "UTAH", IsCmdOrRpt: true}}
	result := AX25_frame{Source_addr: src, Dest_addr: dst, Digi_path: digi}.TNC2()
	expected := "KK7EWJ-7>N0CALL-2,WIDE1,UTAH*:"
	if result != expected {
		t.Errorf("Expected %s but got %s", expected, result)
	}
}

func Test_Callsign_String(t *testing.T) {
	call := Callsign{Call: "KK7EWJ", Ssid: 7}
	expected := "KK7EWJ-7"
	if call.String() != expected {
		t.Errorf("Expected %s but got %s", expected, call)
	}
}

func Test_Callsign_AX25Encode(t *testing.T) {
	//KK7EWJ-7
	data := Callsign{Call: "KK7EWJ", Ssid: 7}
	result := data.AX25Encode(AX25_RESERVED_BITS)
	expected := []byte{0x96, 0x96, 0x6e, 0x8a, 0xae, 0x94, 0x6e}
	if !bytes.Equal(result, expected) {
		t.Errorf("  Result: %x\n", result)
		t.Errorf("Expected: %x\n", expected)
	}
}

func Test_Callsign_AX25Encode_short_call(t *testing.T) {
	data := Callsign{Call: "WIDE1", Ssid: 1}
	result := data.AX25Encode(AX25_RESERVED_BITS)
	expect := []byte{0xae, 0x92, 0x88, 0x8a, 0x62, 0x40, 0x62}

	if !bytes.Equal(result, expect) {
		t.Errorf("Expect %x, Got %x", expect, result)
	}
}

func Test_AX25_frame_Bytes(t *testing.T) {
	raw, err := readBytesFromFile("./test.ax25")
	if err != nil {
		t.Fatal(err)
	}
	expected, err := StripKISSWrapper(raw)
	if err != nil {
		t.Fatal(err)
	}

	// KK7EWJ-7>APDR16,WIDE1-1,WIDE2-1:=3707.62N/11337.37W[/A=002735 hiking
	input := AX25_frame{
		Dest_addr:   Callsign{Call: "APDR16"},
		Source_addr: Callsign{Call: "KK7EWJ", Ssid: 7},
		Digi_path:   []Callsign{{Call: "WIDE1", Ssid: 1}, {Call: "WIDE2", Ssid: 1}},
		Info_field:  "=3707.62N/11337.37W[/A=002735 hiking",
	}
	result := input.Bytes(false)

	if !bytes.Equal(result, expected) {
		t.Logf("exp: %x \n", expected[14:21]) //
		t.Logf("res: %x \n", result[14:21])   //
		t.Error("Do not match")
	}
}
