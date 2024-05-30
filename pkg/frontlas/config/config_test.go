package config

import (
	"os"
	"testing"
)

func TestGenDefaultConfig(t *testing.T) {
	file, err := os.OpenFile("../../../etc/frontlas.yaml", os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		t.Error(err)
	}
	defer file.Close()
	err = genMinConfig(file)
	if err != nil {
		t.Error(err)
	}
}

func TestGenAllConfig(t *testing.T) {
	file, err := os.OpenFile("../../../etc/frontlas_all.yaml", os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		t.Error(err)
	}
	defer file.Close()
	err = genAllConfig(file)
	if err != nil {
		t.Error(err)
	}
}
