package main

import (
	"bytes"
	"internal/AX25"
	"net"
	"testing"
	"time"
)

// emulates a KISS server for testing purposes. Only meant for one connection at a time. Multiple WILL break this.
func internal_KISSServerEmulator(address string) (string, chan []byte, chan error, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return "", nil, nil, err
	}

	input_chan := make(chan []byte)
	shutdown_chan := make(chan struct{})
	err_chan := make(chan error)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-shutdown_chan:
					return
				default:
					err_chan <- err
				}
			}

			go func() {
				for {
					data := &bytes.Buffer{}
					data.Write([]byte{0xc0, 0x0})
					data.Write(<-input_chan)
					data.Write([]byte{0xc0})

					_, err := conn.Write(data.Bytes())
					if err != nil {
						err_chan <- err
						return
					}
				}
			}()

		}
	}()

	return listener.Addr().String(), input_chan, err_chan, nil
}

func Test_KISSServerConnector_failsOnBadTarget(t *testing.T) {

	srv := "badresolver.kk7ewj.net:0000"

	_, _, err := KISSServerConnector(srv)

	if err != nil {
		// expected behavior
		return
	} else {
		t.Errorf("Expected an error to throw for bad srv addr, but got none")
	}

}

func Test_KISSServerConnector_goodConnection(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer func() {
		t.Log("Closing listener")
		listener.Close()
	}()
	srv := listener.Addr().String()
	t.Log(srv)

	_, error_chan, err := KISSServerConnector(srv)

	if err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-error_chan:
		t.Fatal(err)
	case <-time.After(time.Second * 1):
		return // expected behavior
	}

}

func Test_KISServerConnector_various_inputs(t *testing.T) {
	// set up the fake emulator server
	em_addr, em_input_chan, _, err := internal_KISSServerEmulator("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	connector_out_chan, connector_err_chan, err := KISSServerConnector(em_addr)
	if err != nil {
		t.Fatal(err)
	}

	test_data1 := AX25.AX25_frame{
		Dest_addr:   AX25.Callsign{Call: "APZ123"},
		Source_addr: AX25.Callsign{Call: "KK7EWJ", Ssid: 7},
		Info_field:  ":TEST     :Hello World!",
	}.Bytes(false)
	em_input_chan <- test_data1

	select {
	case data := <-connector_out_chan:
		t.Log(data)
	case err := <-connector_err_chan:
		t.Errorf("test: %v", err)
	}

}
