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
