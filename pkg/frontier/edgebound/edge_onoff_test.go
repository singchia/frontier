package edgebound

import (
	"net"
	"testing"
	"time"

	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/frontier/repo"
	"github.com/singchia/geminio"
)

type reconnectEnd struct {
	geminio.End
	id          uint64
	addr        net.Addr
	closeCalled chan struct{}
	closeWait   <-chan struct{}
}

func (end *reconnectEnd) ClientID() uint64     { return end.id }
func (end *reconnectEnd) RemoteAddr() net.Addr { return end.addr }
func (end *reconnectEnd) Meta() []byte         { return []byte("edge") }
func (end *reconnectEnd) Close() error {
	if end.closeCalled != nil {
		close(end.closeCalled)
	}
	if end.closeWait != nil {
		<-end.closeWait
	}
	return nil
}

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
