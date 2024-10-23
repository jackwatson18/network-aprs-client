package main

import (
	"net"
	"testing"
	"time"
)

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

// func Test_KISServerConnector_various_inputs(t *testing.T) {
// 	// set up the fake emulator server
// 	listener, err := net.Listen("tcp", "127.0.0.1:0")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	listen_addr := listener.Addr().String()
// 	data_chan := make(chan []byte)
// 	conn_chan := make(chan net.Conn)

// 	em_done_chan, em_err_chan := KISSServerEmulator(conn_chan, data_chan)

// 	// set up KISSServerConnector
// 	ax25_chan, kiss_err_chan, err := KISSServerConnector(listen_addr)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// do stuff

// }
