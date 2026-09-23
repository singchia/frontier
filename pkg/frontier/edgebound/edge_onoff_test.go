package edgebound

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/frontier/repo"
	"github.com/singchia/frontier/pkg/frontier/repo/query"
	"github.com/singchia/geminio"
)

type reconnectEnd struct {
	geminio.End
	id          uint64
	addr        net.Addr
	closeCalled chan struct{}
	closeWait   <-chan struct{}
	closeOnce   sync.Once
}

func (end *reconnectEnd) ClientID() uint64     { return end.id }
func (end *reconnectEnd) RemoteAddr() net.Addr { return end.addr }
func (end *reconnectEnd) Meta() []byte         { return []byte("edge") }
func (end *reconnectEnd) Close() error {
	if end.closeCalled != nil {
		end.closeOnce.Do(func() { close(end.closeCalled) })
	}
	if end.closeWait != nil {
		<-end.closeWait
	}
	return nil
}

type reconnectStream struct {
	geminio.Stream
	id     uint64
	stream uint64
	addr   net.Addr
}

func (stream *reconnectStream) ClientID() uint64     { return stream.id }
func (stream *reconnectStream) StreamID() uint64     { return stream.stream }
func (stream *reconnectStream) RemoteAddr() net.Addr { return stream.addr }
func (stream *reconnectStream) Meta() []byte         { return nil }

func TestEdgeReOnline_WhenOldSessionDoesNotGoOffline(t *testing.T) {
	r, err := repo.NewRepo(&config.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	em := &edgeManager{edges: make(map[uint64]*edgeSession), repo: r}
	old := &reconnectEnd{id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10001}}
	newEnd := &reconnectEnd{id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10002}}
	oldSession, newSession := &edgeSession{edgeManager: em}, &edgeSession{edgeManager: em}
	if err := em.online(oldSession, old); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() { done <- em.online(newSession, newEnd) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		if err := oldSession.ConnOffline(old); err != nil {
			t.Logf("cleanup old session: %v", err)
		}
		<-done
		t.Fatal("reconnection waited for old session to go offline")
	}
	if em.GetEdgeByID(old.id) != newEnd {
		t.Fatal("new session did not replace the old session")
	}
	newest := &reconnectEnd{id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10003}}
	if err := em.online(&edgeSession{edgeManager: em}, newest); err != nil {
		t.Fatalf("second reconnect was rejected: %v", err)
	}
	if err := oldSession.ConnOffline(old); err != nil {
		t.Fatal(err)
	}
	if err := newSession.ConnOffline(newEnd); err != nil {
		t.Fatal(err)
	}
	if em.GetEdgeByID(old.id) != newest {
		t.Fatal("late old-session callback removed the latest session")
	}
	edge, err := r.GetEdge(old.id)
	if err != nil || edge.Addr != newest.addr.String() {
		t.Fatalf("repository lost the latest session: edge=%+v err=%v", edge, err)
	}
}

func TestEdgeReOnline_WhenOldCloseBlocks(t *testing.T) {
	r, err := repo.NewRepo(&config.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	em := &edgeManager{edges: make(map[uint64]*edgeSession), repo: r}
	closeWait := make(chan struct{})
	old := &reconnectEnd{
		id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10001},
		closeCalled: make(chan struct{}), closeWait: closeWait,
	}
	newEnd := &reconnectEnd{id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10002}}
	oldSession, newSession := &edgeSession{edgeManager: em}, &edgeSession{edgeManager: em}
	if err := em.online(oldSession, old); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- em.online(newSession, newEnd) }()
	select {
	case <-old.closeCalled:
	case <-time.After(time.Second):
		close(closeWait)
		t.Fatal("old connection was not closed")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		close(closeWait)
		t.Fatal("reconnection waited for old Close to return")
	}
	close(closeWait)
}

func TestEdgeReOnline_LateOldStreamsDoNotReplaceNewStreams(t *testing.T) {
	r, err := repo.NewRepo(&config.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	em := &edgeManager{edges: make(map[uint64]*edgeSession), repo: r}
	old := &reconnectEnd{id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10001}}
	newEnd := &reconnectEnd{id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10002}}
	oldSession, newSession := &edgeSession{edgeManager: em}, &edgeSession{edgeManager: em}
	if err := em.online(oldSession, old); err != nil {
		t.Fatal(err)
	}
	oldStream := &reconnectStream{id: 72, stream: 3, addr: old.addr}
	oldSession.acceptStream(oldStream)
	newStream := &reconnectStream{id: 72, stream: 3, addr: newEnd.addr}
	newSession.acceptStream(newStream) // The new stream can arrive before online installs its end.
	if streams := em.ListStreams(72); len(streams) != 1 || streams[0] != oldStream {
		t.Fatal("early new stream appeared in the old session")
	}
	if err := em.online(newSession, newEnd); err != nil {
		t.Fatal(err)
	}
	if streams := em.ListStreams(72); len(streams) != 1 || streams[0] != newStream {
		t.Fatal("early new stream was not visible after reconnect")
	}
	oldSession.closedStream(oldStream)
	if streams := em.ListStreams(72); len(streams) != 1 || streams[0] != newStream {
		t.Fatal("old close removed replacement stream")
	}
	lateOldStream := &reconnectStream{id: 72, stream: 3, addr: old.addr}
	oldSession.acceptStream(lateOldStream)
	if streams := em.ListStreams(72); len(streams) != 1 || streams[0] != newStream {
		t.Fatal("late old stream replaced the active stream")
	}
}

func TestEdgeReOnline_ReusedAddressKeepsReplacement(t *testing.T) {
	r, err := repo.NewRepo(&config.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	em := &edgeManager{edges: make(map[uint64]*edgeSession), repo: r}
	addr := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10001}
	old := &reconnectEnd{id: 72, addr: addr}
	newEnd := &reconnectEnd{id: 72, addr: addr}
	oldSession, newSession := &edgeSession{edgeManager: em}, &edgeSession{edgeManager: em}
	if err := em.online(oldSession, old); err != nil {
		t.Fatal(err)
	}
	oldStream := &reconnectStream{id: 72, stream: 3, addr: addr}
	newStream := &reconnectStream{id: 72, stream: 3, addr: addr}
	oldSession.acceptStream(oldStream)
	newSession.acceptStream(newStream)
	if err := em.online(newSession, newEnd); err != nil {
		t.Fatal(err)
	}
	oldSession.closedStream(oldStream)
	if streams := em.ListStreams(72); len(streams) != 1 || streams[0] != newStream {
		t.Fatal("old stream cleanup removed the replacement stream at the same address")
	}
	if err := oldSession.ConnOffline(old); err != nil {
		t.Fatal(err)
	}
	if em.GetEdgeByID(72) != newEnd {
		t.Fatal("old callback removed a replacement that reused the remote address")
	}
}

func TestEdgeReOnline_OfflineBeforeInstallationDoesNotLeaveStaleSession(t *testing.T) {
	r, err := repo.NewRepo(&config.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	em := &edgeManager{edges: make(map[uint64]*edgeSession), repo: r}
	session := &edgeSession{edgeManager: em}
	end := &reconnectEnd{id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10001}}
	if err := session.ConnOffline(end); err != nil {
		t.Fatal(err)
	}
	if err := em.online(session, end); err != net.ErrClosed {
		t.Fatalf("closed session was installed: %v", err)
	}
	if em.GetEdgeByID(72) != nil {
		t.Fatal("closed session is still active")
	}
}

func TestEdgeReOnline_RPCsBelongToCurrentSession(t *testing.T) {
	for _, backend := range []string{"buntdb", "sqlite3"} {
		t.Run(backend, func(t *testing.T) {
			conf := &config.Configuration{}
			conf.Dao.Backend = backend
			r, err := repo.NewRepo(conf)
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			em := &edgeManager{edges: make(map[uint64]*edgeSession), repo: r}
			old := &reconnectEnd{id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10001}}
			newEnd := &reconnectEnd{id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10002}}
			oldSession, newSession := &edgeSession{edgeManager: em}, &edgeSession{edgeManager: em}
			if err := em.online(oldSession, old); err != nil {
				t.Fatal(err)
			}
			oldSession.RemoteRegistration("old_method", 72, 1)
			newSession.RemoteRegistration("new_method", 72, 1)
			if err := em.online(newSession, newEnd); err != nil {
				t.Fatal(err)
			}
			oldSession.RemoteRegistration("late_old_method", 72, 1)
			rpcs, err := r.ListEdgeRPCs(&query.EdgeRPCQuery{EdgeID: 72})
			if err != nil || len(rpcs) != 1 || rpcs[0] != "new_method" {
				t.Fatalf("RPC registry contains stale sessions: rpcs=%v err=%v", rpcs, err)
			}
		})
	}
}

func TestEdgeOffline_CleanupErrorDoesNotKeepClosedSession(t *testing.T) {
	r, err := repo.NewRepo(&config.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	em := &edgeManager{edges: make(map[uint64]*edgeSession), repo: r}
	session := &edgeSession{edgeManager: em}
	end := &reconnectEnd{id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10001}}
	if err := em.online(session, end); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if err := session.ConnOffline(end); err == nil {
		t.Fatal("expected the repository cleanup to fail")
	}
	if em.GetEdgeByID(72) != nil {
		t.Fatal("repository error left a closed connection routable")
	}
}

func TestDelEdgeByID_CloseDoesNotBlockReconnect(t *testing.T) {
	r, err := repo.NewRepo(&config.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	em := &edgeManager{edges: make(map[uint64]*edgeSession), repo: r}
	closeWait := make(chan struct{})
	old := &reconnectEnd{
		id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10001},
		closeCalled: make(chan struct{}), closeWait: closeWait,
	}
	newEnd := &reconnectEnd{id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10002}}
	oldSession, newSession := &edgeSession{edgeManager: em}, &edgeSession{edgeManager: em}
	if err := em.online(oldSession, old); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- em.DelEdgeByID(72) }()
	select {
	case <-old.closeCalled:
	case <-time.After(time.Second):
		close(closeWait)
		t.Fatal("old close was not called")
	}
	onlineDone := make(chan error, 1)
	go func() { onlineDone <- em.online(newSession, newEnd) }()
	select {
	case err := <-onlineDone:
		if err != nil {
			close(closeWait)
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		close(closeWait)
		<-onlineDone
		t.Fatal("reconnection waited for management Close to return")
	}
	close(closeWait)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
