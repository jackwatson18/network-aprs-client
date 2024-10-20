package main

import (
	"testing"
)

func Test_test(t *testing.T) {
	if 1+1 != 2 {
		t.Error("Something has gone horribly wrong...")
	}
}
