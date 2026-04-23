package frame_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/singchia/frontier/examples/wireguard/internal/frame"
)

func TestRoundTrip(t *testing.T) {
	payloads := [][]byte{
		[]byte("hello"),
		[]byte("wireguard handshake initiation"),
		bytes.Repeat([]byte{0xAB}, 1420),
	}
	buf := &bytes.Buffer{}
	for _, p := range payloads {
		if err := frame.WriteFrame(buf, p); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	for i, want := range payloads {
		got, err := frame.ReadFrame(buf)
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("frame %d mismatch: got %d bytes, want %d", i, len(got), len(want))
		}
	}
}

func TestMaxPayload(t *testing.T) {
	p := bytes.Repeat([]byte{0xCD}, 65535)
	buf := &bytes.Buffer{}
	if err := frame.WriteFrame(buf, p); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := frame.ReadFrame(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, p) {
		t.Fatal("max-payload mismatch")
	}
}

func TestOversizedPayloadRejected(t *testing.T) {
	p := bytes.Repeat([]byte{0x00}, 65536)
	buf := &bytes.Buffer{}
	err := frame.WriteFrame(buf, p)
	if err == nil {
		t.Fatal("expected error for oversized payload")
	}
	if buf.Len() != 0 {
		t.Fatalf("oversized payload wrote %d bytes; expected 0", buf.Len())
	}
}

func TestEmptyPayloadRejected(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := frame.WriteFrame(buf, nil); err == nil {
		t.Fatal("expected error for nil payload")
	}
	if err := frame.WriteFrame(buf, []byte{}); err == nil {
		t.Fatal("expected error for empty payload")
	}
	if buf.Len() != 0 {
		t.Fatalf("empty payload wrote %d bytes; expected 0", buf.Len())
	}
}

func TestZeroLengthOnWireRejected(t *testing.T) {
	r := bytes.NewReader([]byte{0x00, 0x00})
	_, err := frame.ReadFrame(r)
	if err == nil {
		t.Fatal("expected error for zero-length frame")
	}
}

func TestShortReadIsUnexpectedEOF(t *testing.T) {
	good := &bytes.Buffer{}
	if err := frame.WriteFrame(good, []byte("hello world")); err != nil {
		t.Fatalf("setup: %v", err)
	}
	truncated := good.Bytes()[:good.Len()-3]
	_, err := frame.ReadFrame(bytes.NewReader(truncated))
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("expected ErrUnexpectedEOF, got %v", err)
	}
}

func TestEOFBeforeHeaderIsEOF(t *testing.T) {
	_, err := frame.ReadFrame(bytes.NewReader(nil))
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}
