/*
This is an implementation of the power of 2 choices selction method, made popular in this paper:
https://www.eecs.harvard.edu/~michaelm/postscripts/mythesis.pdf

This paper is based on the work of:
https://homes.cs.washington.edu/~karlin/papers/AzarBKU99.pdf
*/

package http

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/8/rollout/lb/proto"
)

// weightedBackend implements Backend with a wrapper around another Backend. This
// implementation adds a weight that can be used to help determine which Backend to
// choose when doing a P2C.
type weightedBackend struct {
	Backend

	weight int32
}

func (w *weightedBackend) get() int32 {
	return atomic.LoadInt32(&w.weight)
}

func (w *weightedBackend) call() {
	w.Backend.call()
	atomic.AddInt32(&w.weight, 1)
}

func (w *weightedBackend) done() {
	w.Backend.done()
	i := atomic.AddInt32(&w.weight, -1)
	if i < 0 {
		panic("weightedBackend cannot be < 0")
	}
}

func (w *weightedBackend) handler() http.Handler {
	return http.HandlerFunc(
		func(wr http.ResponseWriter, r *http.Request) {
			w.call()
			defer w.done()
			w.Backend.handler().ServeHTTP(wr, r)
		},
	)
}

// P2C implements Pool using the Power of 2 choice selection method.
type P2C struct {
	hc       HealthCheck
	interval time.Duration

	mu            sync.Mutex
	healthy, sick *atomic.Value // []*weightedBackend
	rand          *rand.Rand

	done chan struct{}
}

// NewP2C creates a new P2C instance. hc is the health check
// to perform on the backend to make sure its healthy and interval is how often to do
// the health check.
func NewP2C(hc HealthCheck, interval time.Duration) (*P2C, error) {
	sp := &P2C{
		hc:       hc,
		interval: interval,
		healthy:  &atomic.Value{},
		sick:     &atomic.Value{},
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
		done:     make(chan struct{}),
	}

	sp.healthy.Store([]*weightedBackend{})
	sp.sick.Store([]*weightedBackend{})
	go sp.healthLoop()

	return sp, nil
}

// Close implements Pool.Close().
func (s *P2C) Close() error {
	close(s.done)
	return nil
}

// Add implements Pool.Add().
func (s *P2C) Add(ctx context.Context, b Backend) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if err := s.hc(ctx, b.url().String()); err != nil {
		b.setHealth(sick)
		return fmt.Errorf("backend is sick: %w", err)
	}
	b.setHealth(healthy)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.addToValue(&weightedBackend{Backend: b}, s.healthy); err != nil {
		return err
	}

	return nil
}

// Remove implements Pool.Remove().
func (s *P2C) Remove(ctx context.Context, b Backend) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.removeFromValue(b, s.healthy)
	s.removeFromValue(b, s.sick)
	return nil
}

// Health implements Pool.Health().
func (s *P2C) Health(ctx context.Context, req *pb.PoolHealthReq) (*pb.PoolHealth, error) {
	status := pb.PoolStatus_PS_FULL

	healthy := s.healthy.Load().([]*weightedBackend)
	sick := s.sick.Load().([]*weightedBackend)

	healthyNodes := len(healthy)
	sickNodes := len(sick)

	if sickNodes == 0 && healthyNodes == 0 {
		return &pb.PoolHealth{
			Status: pb.PoolStatus_PS_EMPTY,
		}, nil
	}

	if sickNodes > 0 {
		status = pb.PoolStatus_PS_DEGRADED
	}

	ph := &pb.PoolHealth{
		Status: status,
	}

	if req.Healthy {
		for _, wb := range healthy {
			switch v := wb.Backend.(type) {
			case *IPBackend:
				h := &pb.BackendHealth{
					Status: pb.BackendStatus_BS_HEALTHY,
					Backend: &pb.Backend{
						Backend: &pb.Backend_IpBackend{
							IpBackend: &pb.IPBackend{
								Ip:      v.ip.String(),
								Port:    v.port,
								UrlPath: v.urlPath,
							},
						},
					},
				}
				ph.Backends = append(ph.Backends, h)
			default:
				return nil, fmt.Errorf("an unknown healthy backend type found(%T)", wb.Backend)
			}
		}
	}
	if req.Sick {
		for _, wb := range sick {
			switch v := wb.Backend.(type) {
			case *IPBackend:
				h := &pb.BackendHealth{
					Status: pb.BackendStatus_BS_SICK,
					Backend: &pb.Backend{
						Backend: &pb.Backend_IpBackend{
							IpBackend: &pb.IPBackend{
								Ip:      v.ip.String(),
								Port:    v.port,
								UrlPath: v.urlPath,
							},
						},
					},
				}
				ph.Backends = append(ph.Backends, h)
			default:
				return nil, fmt.Errorf("an unknown sick backend type found(%T)", wb.Backend)
			}
		}
	}
	return ph, nil
}

func (s *P2C) addToValue(b *weightedBackend, v *atomic.Value) error {
	backs := (*v).Load().([]*weightedBackend)
	n := make([]*weightedBackend, 0, len(backs)+1)
	for _, back := range backs {
		// This is quite slow, but... we should not be adding backends often
		// to a single instance. If this somehow becomes a bottleneck, we can
		// always calculate some hash on Add() to do checks on.
		if b.url().String() == back.url().String() {
			return fmt.Errorf("backend already exists")
		}
		n = append(n, back)
	}
	n = append(n, b)

	v.Store(n)
	return nil
}

func (s *P2C) removeFromValue(b Backend, v *atomic.Value) error {
	backs := v.Load().([]*weightedBackend)

	newCap := len(backs) - 1
	if newCap < 0 {
		return nil // No way it exists
	}

	n := make([]*weightedBackend, 0, newCap)
	for _, back := range backs {
		if b.url().String() == back.url().String() {
			continue
		}
		n = append(n, back)
	}
	if len(backs) == len(n) {
		return fmt.Errorf("could not find backend(%s)", b.url().String())
	}

	v.Store(n)
	return nil
}

// ServeHTTP implements Pool.ServeHTTP().
func (s *P2C) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backs := s.healthy.Load().([]*weightedBackend)
	if len(backs) == 0 {
		http.Error(w, "no backends available", http.StatusInternalServerError)
		return
	}
	x := s.rand.Int31n(int32(len(backs)))
	y := s.rand.Int31n(int32(len(backs)))

	if backs[x].weight < backs[y].weight {
		backs[x].handler().ServeHTTP(w, r)
		return
	}
	backs[y].handler().ServeHTTP(w, r)
}

func (s *P2C) healthLoop() {
	for {
		select {
		case <-s.done:
			return
		case <-time.After(s.interval):
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			s.healthChecks(ctx)
			cancel()
		}
	}
}

func (s *P2C) healthChecks(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wg := sync.WaitGroup{}
	for _, b := range s.healthy.Load().([]*weightedBackend) {
		b := b
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.hc(ctx, b.url().String()); err != nil {
				s.healthyToSick(b)
			}
		}()
	}
	for _, b := range s.sick.Load().([]*weightedBackend) {
		b := b
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.hc(ctx, b.url().String()); err == nil {
				s.sickToHealthy(b)
			}
		}()
	}
	wg.Wait()
}

func (s *P2C) healthyToSick(b *weightedBackend) {
	log.Printf("backend %s became sick", b.url())
	b.setHealth(sick)
	if err := s.removeFromValue(b, s.healthy); err != nil {
		log.Println(err)
		return
	}
	if err := s.addToValue(b, s.sick); err != nil {
		panic(err)
	}
}

func (s *P2C) sickToHealthy(b *weightedBackend) {
	log.Printf("backend %s became healthy", b.url())
	b.setHealth(healthy)
	if err := s.removeFromValue(b, s.sick); err != nil {
		log.Println(err)
		return
	}
	if err := s.addToValue(b, s.healthy); err != nil {
		panic(err)
	}
}
