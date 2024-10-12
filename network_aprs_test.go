package main

import (
	"testing"
)

func Test_HelloWorld(t *testing.T) {
	t.Log("Hello world, testing!")
}

func Test_ReadBytesFromFile_validFile(t *testing.T) {
	_, err := ReadBytesFromFile("./packetFiles/test.ax25")

	if err != nil {
		t.Fatalf(`%v`, err)
	}

}

func Test_ReadBytesFromFile_NonexistantFile(t *testing.T) {
	_, err := ReadBytesFromFile("./badDirectory/badFile.ax25")

	if err == nil {
		t.Errorf("Expected an error, but ReadBytesFromFile did not produce error")
	}

}

func Test_stripKISSWrapper_FailIfTooShort(t *testing.T) {
	data := make([]byte, 3)
	_, err := stripKISSWrapper(data)

	if err == nil {
		t.Errorf("Expected an error about input too short, but stripKISSWrapper generated no error")
	}
}

func Test_stripKISSWrapper_FailFirstNotFEND(t *testing.T) {
	data := make([]byte, 4)
	_, err := stripKISSWrapper(data)

	if err == nil {
		t.Errorf("Expected an error about no FEND wrapper, but stripKISSWrapper generated no error")
	}
}

func Test_stripKISSWrapper_FailLastNotFEND(t *testing.T) {
	data := make([]byte, 4)
	data[0] = FEND
	_, err := stripKISSWrapper(data)

	if err == nil {
		t.Errorf("Expected an error about no FEND wrapper, but stripKISSWrapper generated no error")
	}
}

func Test_stripKISSWrapper_Succeed(t *testing.T) {
	data := make([]byte, 4)
	data[0] = FEND
	data[3] = FEND
	_, err := stripKISSWrapper(data)

	if err != nil {
		t.Fatalf(`%v`, err)
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
	raw, err := ReadBytesFromFile("./packetFiles/test.ax25")
	if err != nil {
		t.Fatalf(`%v`, err)
	}

	ax25_raw, err := stripKISSWrapper(raw)
	if err != nil {
		t.Fatalf(`%v`, err)
	}

	_, err = ConvertBytesToAX25(ax25_raw)
	if err != nil {
		t.Fatalf(`%v`, err)
	}

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

func Test_parseAddr_CorrectOutput(t *testing.T) {
	result, err := parseAddr([]byte{0x96, 0x96, 0x6E, 0x8A, 0xAE, 0x94, 0x6F})
	expect := "KK7EWJ-7"

	if err != nil {
		t.Errorf("Expected no error but got %v", err)
	}

	if result != expect {
		t.Errorf("Expected %v but got %v", expect, result)
	}
}

func Test_AX25_frame_String(t *testing.T) {
	input := AX25_frame{Dest_addr: "KK7EWJ-0", Source_addr: "N0CALL-0", Digi_path: []string{"WIDE1-1", "WIDE2-1"}, Info_field: "foobar"}.String()
	expect := `      Dest: KK7EWJ-0
    Source: N0CALL-0
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
