package config

import (
	"os"
	"testing"
)

func TestGenDefaultConfig(t *testing.T) {
	file, err := os.OpenFile("../../../etc/frontlas.yaml", os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		t.Error(err)
	}
	defer file.Close()
	err = genDefaultConfig(file)
	if err != nil {
		t.Error(err)
	}
}
