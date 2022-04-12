package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/8/rollout/lb/proto"
)

type routeHandler struct {
	muxStore atomic.Value // *http.ServeMux
}

func newRouteHandler(mux *http.ServeMux) *routeHandler {
	if mux == nil {
		panic("mux cannot be nil")
	}
	r := &routeHandler{}
	r.muxStore.Store(mux)
	return r
}

func (r *routeHandler) mux() *http.ServeMux {
	return r.muxStore.Load().(*http.ServeMux)
}

func (r *routeHandler) replace(mux *http.ServeMux) {
	if mux == nil {
		return
	}
	r.muxStore.Store(mux)
}

func (r *routeHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.muxStore.Load().(http.Handler).ServeHTTP(w, req)
}

func newServ(handler *routeHandler) *http.Server {
	return &http.Server{
		Handler:        handler,
		IdleTimeout:    5 * time.Second,
		MaxHeaderBytes: 1024 * 1000,
	}
}

// LoadBalancer is an HTTP reverse proxy load balancer.
type LoadBalancer struct {
	mu      sync.Mutex
	pools   map[string]Pool
	handler *routeHandler
	serv    *http.Server
}

// New creates a new LoadBalancer instance.
func New() (*LoadBalancer, error) {
	handler := newRouteHandler(http.NewServeMux())
	return &LoadBalancer{
		pools:   map[string]Pool{},
		handler: handler,
		serv:    newServ(handler),
	}, nil
}

// AddPool adds a pool of backends that serve the serveURL listed here. If a pattern
// is added more than once, this will panic.
func (l *LoadBalancer) AddPool(pattern string, pool Pool) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.pools[pattern]; ok {
		return fmt.Errorf("pattern(%s) is already registered", pattern)
	}
	l.pools[pattern] = pool

	l.handler.mux().Handle(pattern, pool)
	return nil
}

// GetPool returns a pool by its pattern.
func (l *LoadBalancer) GetPool(pattern string) (Pool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	p, ok := l.pools[pattern]
	if ok {
		return p, nil
	}
	return nil, fmt.Errorf("pool(%s) not found", pattern)
}

// RemovePool removes a pool of backends that serve a pattern. If the pattern does not
// exist, the error is still nil.
func (l *LoadBalancer) RemovePool(pattern string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	p, ok := l.pools[pattern]
	if !ok {
		return nil
	}
	p.Close()

	delete(l.pools, pattern)

	mux := http.NewServeMux()
	for k, v := range l.pools {
		mux.Handle(k, v)
	}

	l.handler.replace(mux)
	return nil
}

// PoolHealth returns the health of a pool as defined in the req.
func (l *LoadBalancer) PoolHealth(ctx context.Context, req *pb.PoolHealthReq) (*pb.PoolHealth, error) {
	l.mu.Lock()
	p, ok := l.pools[req.Pattern]
	l.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("pool not found")
	}
	return p.Health(ctx, req)
}

// Serve will serve HTTP traffic(non-TLS) on lis.
func (l *LoadBalancer) Serve(lis net.Listener) error {
	return l.serv.Serve(lis)
}

// ServeTLS will serve HTTPS traffic on lis. See http.Server.ServeTLS for more documentation.
func (l *LoadBalancer) ServeTLS(lis net.Listener, certFile, keyFile string) error {
	return l.serv.ServeTLS(lis, certFile, keyFile)
}

// Pool represents a set of backends that serve a URL.
type Pool interface {
	// Add adds a new Backend to the pool. The Backend must be healthy.
	Add(ctx context.Context, b Backend) error
	// Remove removes a backend from the loadbalancer.
	Remove(ctx context.Context, b Backend) error
	// Health returns the health of a pool.
	Health(ctx context.Context, req *pb.PoolHealthReq) (*pb.PoolHealth, error)
	// Close closes the pool. It should not be used after this.
	Close() error
	// ServeHTTP implements http.Handler.
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// HealthCheck provides a health check for a backend. The state passed is the
// current state of the backend. If state is Healthy but fails the check, this
// will cause the backend to be removed from service . If state is Sick and
// passes the check, the backend will be returned to service.
type HealthCheck func(ctx context.Context, endpoint string) error

// HealthMultiplexer allows combining multiple HealthCheck(s) together.
func HealthMultiplexer(healthChecks ...HealthCheck) HealthCheck {
	return func(ctx context.Context, endpoint string) error {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		results := make(chan error, len(healthChecks))
		for _, hc := range healthChecks {
			hc := hc
			go func() {
				results <- hc(ctx, endpoint)
			}()
		}
		for i := 0; i < len(healthChecks); i++ {
			result := <-results
			if result != nil {
				return result
			}
		}
		return nil
	}
}

// StatusCheck returns a HealthCheck that checks the status
func StatusCheck(urlPath string, healthyValues []string) (HealthCheck, error) {
	if len(healthyValues) == 0 {
		return nil, fmt.Errorf("must provide at least one healthy value")
	}
	return func(ctx context.Context, endpoint string) error {
		u, err := url.Parse(endpoint)
		if err != nil {
			return err
		}
		u.Scheme = "http"
		u.Path = urlPath

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		c := &http.Client{}
		req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		if err != nil {
			return err
		}
		resp, err := c.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		b = bytes.TrimSpace(b)
		for _, v := range healthyValues {
			if len(v) != len(b) {
				continue
			}
			if string(b) == v {
				return nil
			}
		}
		return fmt.Errorf("not healthy, got status(%s)", b)
	}, nil
}

// healthState is the health state of a backend.
type healthState int8

const (
	unknownHS healthState = 0
	healthy   healthState = 1
	sick      healthState = 2
)

type Backend interface {
	// url is the URL of the backend.
	url() *url.URL
	// setSick marks the node as sick.
	setHealth(healthState)
	// health returns the current healthState
	health() healthState
	// call is called before a backend is used.
	call()
	// done is called after a backend is used.
	done()
	// handler provides the backends http.Handler.
	handler() http.Handler
}

// IPBackend provides a backend to our proxy that will use ip:port as the backend.
// This gives us static backends that isolate us from DNS changes or failures.
type IPBackend struct {
	ip      net.IP
	port    int32
	urlPath string

	healthState atomic.Value // HealthState

	endpoint string
	u        *url.URL
	handle   *httputil.ReverseProxy
}

// NewIPBackend is the constructor for IPBackend.
func NewIPBackend(ip net.IP, port int32, urlPath string) (*IPBackend, error) {
	i := &IPBackend{
		ip:       ip,
		port:     port,
		endpoint: fmt.Sprintf("%s:%d", ip, port),
	}
	i.healthState.Store(unknownHS)
	i.resolveURL()
	i.handle = httputil.NewSingleHostReverseProxy(i.u)

	if err := i.validate(); err != nil {
		return nil, err
	}

	return i, nil
}

func (i *IPBackend) validate() error {
	if i.ip.To4() == nil && i.ip.To16() == nil {
		return fmt.Errorf("ip %q was not valid", i.ip)
	}
	if i.port < 1 || i.port > 65534 {
		return fmt.Errorf("port %d was not valid", i.port)
	}
	return nil
}

func (i *IPBackend) url() *url.URL {
	return i.u
}

func (i *IPBackend) setHealth(hs healthState) {
	i.healthState.Store(hs)
}

func (i *IPBackend) health() healthState {
	return i.healthState.Load().(healthState)
}

func (i *IPBackend) call() {} // not needed
func (i *IPBackend) done() {} // not needed

func (i *IPBackend) handler() http.Handler {
	return i.handle
}

func (i *IPBackend) resolveURL() error {
	u, err := url.Parse(i.urlPath)
	if err != nil {
		return err
	}
	base, err := url.Parse("http://" + i.endpoint)
	if err != nil {
		return err
	}
	i.u = base.ResolveReference(u)
	return nil
}
