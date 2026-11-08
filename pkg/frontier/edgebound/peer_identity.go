package edgebound

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strconv"
	"time"
)

func verifiedPeerURIs(conn net.Conn) ([]string, error) {
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return nil, errors.New("TLS identity required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		return nil, err
	}
	state := tlsConn.ConnectionState()
	if len(state.VerifiedChains) == 0 || len(state.PeerCertificates) == 0 {
		return nil, errors.New("verified client certificate required")
	}
	uris := make([]string, 0, len(state.PeerCertificates[0].URIs))
	for _, uri := range state.PeerCertificates[0].URIs {
		uris = append(uris, uri.String())
	}
	return uris, nil
}
func verifyPeerID(prefix string, uris []string, id uint64) error {
	if prefix == "" {
		return nil
	}
	if len(uris) != 1 || uris[0] != prefix+strconv.FormatUint(id, 10) {
		return errors.New("client certificate identity mismatch")
	}
	return nil
}
func (session *edgeSession) GetClientID(proposed uint64, meta []byte) (uint64, error) {
	id, err := session.edgeManager.GetClientID(proposed, meta)
	if err != nil {
		return 0, err
	}
	if err = verifyPeerID(session.conf.Edgebound.Listen.TLS.ClientIdentityURIPrefix, session.peerURIs, id); err != nil {
		return 0, err
	}
	return id, nil
}
