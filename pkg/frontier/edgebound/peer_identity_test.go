package edgebound

import "testing"

func TestPeerCertificateIdentityBinding(t *testing.T) {
	for _, c := range []struct {
		name, prefix string
		uris         []string
		id           uint64
		valid        bool
	}{
		{"legacy", "", nil, 42, true},
		{"matching", "spiffe://example/edge/", []string{"spiffe://example/edge/42"}, 42, true},
		{"different edge", "spiffe://example/edge/", []string{"spiffe://example/edge/41"}, 42, false},
		{"different role", "spiffe://example/edge/", []string{"spiffe://example/device/42"}, 42, false},
		{"ambiguous", "spiffe://example/edge/", []string{"spiffe://example/edge/42", "spiffe://example/edge/43"}, 42, false},
		{"missing", "spiffe://example/edge/", nil, 42, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if err := verifyPeerID(c.prefix, c.uris, c.id); (err == nil) != c.valid {
				t.Fatalf("err=%v", err)
			}
		})
	}
}
