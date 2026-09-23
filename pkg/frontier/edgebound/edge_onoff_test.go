package edgebound

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/frontier/repo"
	"github.com/singchia/frontier/pkg/mapmap"
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

	em := &edgeManager{edges: make(map[uint64]geminio.End), repo: r}
	old := &reconnectEnd{id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10001}}
	newEnd := &reconnectEnd{id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10002}}
	if err := em.online(old); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() { done <- em.online(newEnd) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		if err := em.offline(old.id, old.Meta(), old.addr); err != nil {
			t.Logf("cleanup old session: %v", err)
		}
		<-done
		t.Fatal("reconnection waited for old session to go offline")
	}
	if em.GetEdgeByID(old.id) != newEnd {
		t.Fatal("new session did not replace the old session")
	}
	if err := em.offline(old.id, old.Meta(), old.addr); err != nil {
		t.Fatal(err)
	}
	if em.GetEdgeByID(old.id) != newEnd {
		t.Fatal("late old-session callback removed the new session")
	}
	edge, err := r.GetEdge(old.id)
	if err != nil || edge.Addr != newEnd.addr.String() {
		t.Fatalf("repository lost the new session: edge=%+v err=%v", edge, err)
	}
}

func TestEdgeReOnline_WhenOldCloseBlocks(t *testing.T) {
	r, err := repo.NewRepo(&config.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	em := &edgeManager{edges: make(map[uint64]geminio.End), repo: r}
	closeWait := make(chan struct{})
	old := &reconnectEnd{
		id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10001},
		closeCalled: make(chan struct{}), closeWait: closeWait,
	}
	newEnd := &reconnectEnd{id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10002}}
	if err := em.online(old); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- em.online(newEnd) }()
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

	em := &edgeManager{edges: make(map[uint64]geminio.End), streams: mapmap.NewMapMap(), repo: r}
	old := &reconnectEnd{id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10001}}
	newEnd := &reconnectEnd{id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10002}}
	if err := em.online(old); err != nil {
		t.Fatal(err)
	}
	oldStream := &reconnectStream{id: 72, stream: 3, addr: old.addr}
	em.acceptStream(oldStream)
	if err := em.online(newEnd); err != nil {
		t.Fatal(err)
	}
	newStream := &reconnectStream{id: 72, stream: 3, addr: newEnd.addr}
	em.acceptStream(newStream)
	if got := em.streams.MGet(uint64(72), uint64(3)); got != newStream {
		t.Fatalf("new stream was not cached: got=%v", got)
	}
	em.closedStream(oldStream)
	if got := em.streams.MGet(uint64(72), uint64(3)); got != newStream {
		t.Fatalf("old close removed replacement stream: %v", got)
	}
	lateOldStream := &reconnectStream{id: 72, stream: 3, addr: old.addr}
	em.acceptStream(lateOldStream)
	if em.streams.MGet(uint64(72), uint64(3)) != newStream {
		t.Fatal("late old stream replaced the active stream")
	}
}

func TestDelEdgeByID_CloseDoesNotBlockReconnect(t *testing.T) {
	r, err := repo.NewRepo(&config.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	em := &edgeManager{edges: make(map[uint64]geminio.End), repo: r}
	closeWait := make(chan struct{})
	old := &reconnectEnd{
		id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10001},
		closeCalled: make(chan struct{}), closeWait: closeWait,
	}
	newEnd := &reconnectEnd{id: 72, addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 10002}}
	if err := em.online(old); err != nil {
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
	go func() { onlineDone <- em.online(newEnd) }()
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
