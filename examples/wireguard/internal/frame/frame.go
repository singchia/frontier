// Package frame implements the 2-byte big-endian length-prefix wire format
// used by the WireGuard-over-frontier example. A frame is:
//
//	[2 bytes big-endian length N] [N bytes payload]
//
// N == 0 is a protocol error on the wire. Payloads larger than 65535 bytes
// cannot be represented and are rejected at write time.
package frame

import (
	"encoding/binary"
	"errors"
	"io"
)

// MaxPayloadSize is the largest single frame payload the wire format can carry.
const MaxPayloadSize = 65535

var (
	ErrEmptyPayload    = errors.New("frame: empty payload")
	ErrPayloadTooLarge = errors.New("frame: payload exceeds 65535 bytes")
	ErrZeroLength      = errors.New("frame: zero-length frame on wire")
)

// WriteFrame writes p as a single length-prefixed frame. Header and payload
// are combined into one Write call so a datagram-preserving transport (pion
// UDP) maps each frame to exactly one datagram.
func WriteFrame(w io.Writer, p []byte) error {
	if len(p) == 0 {
		return ErrEmptyPayload
	}
	if len(p) > MaxPayloadSize {
		return ErrPayloadTooLarge
	}
	buf := make([]byte, 2+len(p))
	binary.BigEndian.PutUint16(buf[:2], uint16(len(p)))
	copy(buf[2:], p)
	_, err := w.Write(buf)
	return err
}

// ReadFrame reads one length-prefixed frame. It returns io.EOF if the stream
// is closed cleanly before any header bytes arrive, and io.ErrUnexpectedEOF
// if the stream closes mid-frame.
func ReadFrame(r io.Reader) ([]byte, error) {
	var hdr [2]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint16(hdr[:])
	if n == 0 {
		return nil, ErrZeroLength
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
