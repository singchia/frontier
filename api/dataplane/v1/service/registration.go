package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/singchia/geminio"
)

type registrationTask struct {
	rpc        geminio.RPC
	generation uint64
	running    bool
	next       time.Time
	failures   int
}
type registrationAttempt struct {
	peer       *serviceEnd
	method     string
	task       *registrationTask
	rpc        geminio.RPC
	generation uint64
}

func (end *clusterServiceEnd) enqueue(peer *serviceEnd, method string, rpc geminio.RPC) {
	if end.pending == nil {
		end.pending = make(map[*serviceEnd]map[string]*registrationTask)
	}
	if end.pending[peer] == nil {
		end.pending[peer] = make(map[string]*registrationTask)
	}
	task := end.pending[peer][method]
	if task == nil {
		task = &registrationTask{}
		end.pending[peer][method] = task
	}
	task.rpc = rpc
	task.generation++
	task.next = time.Time{}
	task.failures = 0
}
func (end *clusterServiceEnd) Register(ctx context.Context, method string, rpc geminio.RPC) error {
	if method == "" || rpc == nil {
		return errors.New("RPC name and callback are required")
	}
	select {
	case <-end.closed:
		return errors.New("service closed")
	default:
	}
	end.registrationMtx.Lock()
	end.appMtx.Lock()
	if end.rpcs == nil {
		end.rpcs = make(map[string]geminio.RPC)
	}
	end.rpcs[method] = rpc
	end.appMtx.Unlock()
	end.frontiers.Range(func(_, v interface{}) bool { end.enqueue(v.(*frontierNend).end, method, rpc); return true })
	end.registrationMtx.Unlock()
	return end.reconcileRegistrations(ctx, true)
}

// publishFrontier closes the gap between connection setup and concurrent Register.
func (end *clusterServiceEnd) publishFrontier(id interface{}, peer *frontierNend) (interface{}, bool) {
	end.registrationMtx.Lock()
	defer end.registrationMtx.Unlock()
	prev, ok := end.frontiers.Swap(id, peer)
	end.appMtx.RLock()
	for method, rpc := range end.rpcs {
		end.enqueue(peer.end, method, rpc)
	}
	end.appMtx.RUnlock()
	return prev, ok
}
func (end *clusterServiceEnd) retryRegistrations() {
	parent := end.workerCtx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	// Failures remain in the queue with per-connection backoff.
	if err := end.reconcileRegistrations(ctx, false); err != nil && end.logger != nil {
		end.logger.Warnf("RPC registrations pending: %s", err)
	}
}
func (end *clusterServiceEnd) reconcileRegistrations(ctx context.Context, force bool) error {
	live := make(map[*serviceEnd]bool)
	end.frontiers.Range(func(_, v interface{}) bool { live[v.(*frontierNend).end] = true; return true })
	end.registrationMtx.Lock()
	jobs := []registrationAttempt{}
	for peer, methods := range end.pending {
		if !live[peer] {
			delete(end.pending, peer)
			continue
		}
		for method, task := range methods {
			if task.running || (!force && time.Now().Before(task.next)) {
				continue
			}
			task.running = true
			jobs = append(jobs, registrationAttempt{peer, method, task, task.rpc, task.generation})
		}
	}
	end.registrationMtx.Unlock()
	// Bound registration IO concurrency. A failed peer does not skip other peers.
	work := make(chan registrationAttempt, len(jobs))
	for _, job := range jobs {
		work <- job
	}
	close(work)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error
	for i := 0; i < 8 && i < len(jobs); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range work {
				err := func() (err error) {
					defer func() {
						if recover() != nil {
							err = errors.New("registration panic")
						}
					}()
					call, cancel := context.WithTimeout(ctx, 5*time.Second)
					defer cancel()
					return job.peer.Register(call, job.method, job.rpc)
				}()
				end.registrationMtx.Lock()
				job.task.running = false
				if job.task.generation == job.generation {
					if err == nil {
						delete(end.pending[job.peer], job.method)
					} else {
						job.task.failures++
						shift := job.task.failures - 1
						if shift > 5 {
							shift = 5
						}
						delay := time.Second * time.Duration(1<<shift)
						if delay > 30*time.Second {
							delay = 30 * time.Second
						}
						job.task.next = time.Now().Add(delay * time.Duration(800+rand.Intn(401)) / 1000)
					}
				}
				end.registrationMtx.Unlock()
				if err != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("register %s: %w", job.method, err))
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()
	return errors.Join(errs...)
}
