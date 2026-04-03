package e2e

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2E-MSG-001: Edge publishes to a topic, Service registered on that topic receives it
func TestEdgePublishToService(t *testing.T) {
	const topic = "news"
	received := make(chan []byte, 1)

	svc := newService(t,
		service.OptionServiceName("subscriber"),
		service.OptionServiceReceiveTopics([]string{topic}),
	)
	go func() {
		msg, err := svc.Receive(context.TODO())
		if err == nil {
			received <- msg.Data()
			msg.Done()
		}
	}()

	time.Sleep(30 * time.Millisecond)

	e := newEdge(t)
	payload := []byte("breaking-news")
	msg := e.NewMessage(payload)
	err := e.Publish(context.TODO(), topic, msg)
	require.NoError(t, err)

	select {
	case data := <-received:
		assert.Equal(t, payload, data)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}

// E2E-MSG-002: Service publishes a message to a specific edgeID, Edge receives it
func TestServicePublishToEdge(t *testing.T) {
	received := make(chan []byte, 1)

	e := newEdge(t)
	go func() {
		msg, err := e.Receive(context.TODO())
		if err == nil {
			received <- msg.Data()
			msg.Done()
		}
	}()

	time.Sleep(30 * time.Millisecond)

	svc := newService(t, service.OptionServiceName("publisher"))
	payload := []byte("hello-edge")
	msg := svc.NewMessage(payload)
	err := svc.Publish(context.TODO(), e.EdgeID(), msg)
	require.NoError(t, err)

	select {
	case data := <-received:
		assert.Equal(t, payload, data)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}

// E2E-MSG-003: Multiple services on different topics; messages route correctly
func TestMessageTopicRoute(t *testing.T) {
	topics := []string{"topic-a", "topic-b"}
	receivedA := make(chan []byte, 1)
	receivedB := make(chan []byte, 1)

	svcA := newService(t,
		service.OptionServiceName("svc-a"),
		service.OptionServiceReceiveTopics([]string{topics[0]}),
	)
	svcB := newService(t,
		service.OptionServiceName("svc-b"),
		service.OptionServiceReceiveTopics([]string{topics[1]}),
	)
	go func() {
		if msg, err := svcA.Receive(context.TODO()); err == nil {
			receivedA <- msg.Data()
			msg.Done()
		}
	}()
	go func() {
		if msg, err := svcB.Receive(context.TODO()); err == nil {
			receivedB <- msg.Data()
			msg.Done()
		}
	}()

	time.Sleep(30 * time.Millisecond)

	e := newEdge(t)
	msgA := e.NewMessage([]byte("for-a"))
	msgB := e.NewMessage([]byte("for-b"))
	require.NoError(t, e.Publish(context.TODO(), topics[0], msgA))
	require.NoError(t, e.Publish(context.TODO(), topics[1], msgB))

	for _, ch := range []struct {
		ch      chan []byte
		want    string
		timeout time.Duration
	}{
		{receivedA, "for-a", 3 * time.Second},
		{receivedB, "for-b", 3 * time.Second},
	} {
		select {
		case data := <-ch.ch:
			assert.Equal(t, []byte(ch.want), data)
		case <-time.After(ch.timeout):
			t.Fatalf("timed out waiting for message on topic")
		}
	}
}

// E2E-MSG-004: Edge publishes to a topic with no subscriber => error
func TestMessageTopicNotFound(t *testing.T) {
	e := newEdge(t)
	msg := e.NewMessage([]byte("orphan"))
	err := e.Publish(context.TODO(), "no-such-topic", msg)
	assert.Error(t, err)
}

// E2E-MSG-005: 10 edges publish concurrently, service receives all messages
func TestMessageConcurrent(t *testing.T) {
	const (
		topic   = "concurrent-topic"
		workers = 10
	)
	var mu sync.Mutex
	received := 0
	allDone := make(chan struct{})

	svc := newService(t,
		service.OptionServiceName("concurrent-sub"),
		service.OptionServiceReceiveTopics([]string{topic}),
	)
	go func() {
		for {
			msg, err := svc.Receive(context.TODO())
			if err != nil {
				return
			}
			msg.Done()
			mu.Lock()
			received++
			if received == workers {
				close(allDone)
			}
			mu.Unlock()
		}
	}()

	time.Sleep(30 * time.Millisecond)

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			e := newEdge(t)
			msg := e.NewMessage([]byte("concurrent"))
			if err := e.Publish(context.TODO(), topic, msg); err != nil {
				t.Errorf("publish error: %v", err)
			}
		}()
	}
	wg.Wait()

	select {
	case <-allDone:
		mu.Lock()
		assert.Equal(t, workers, received)
		mu.Unlock()
	case <-time.After(5 * time.Second):
		mu.Lock()
		t.Fatalf("timed out: only received %d/%d messages", received, workers)
		mu.Unlock()
	}
}
