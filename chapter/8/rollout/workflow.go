package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/rollout/lb/client"

	"github.com/fatih/color"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/8/rollout/lb/proto"
)

//go:generate stringer -type=endState
// endStates are the final states after a run of a workflow.
type endState int8

const (
	// esUnknown indicates we haven't reached and end state.
	esUnknown endState = 0
	// esSuccess means that the workflow has completed successfully. This
	// does not mean there haven't been failurea.
	esSuccess endState = 1
	// esPreconditionFailure means no work was done as we failed on a precondition.
	esPreconditionFailure endState = 2
	// esCanaryFailure indicates one of the canaries failed, stopping the workflow.
	esCanaryFailure endState = 3
	// esMaxFailures indicates that the workflow passed the canary phase, but failed
	// at a later phase.
	esMaxFailures endState = 4
)

// workflow represents our rollout workflow.
type workflow struct {
	config *config
	lb     *client.Client

	failures int32
	endState endState

	actions []*actions
}

// newWorkflow creates a new workflow.
func newWorkflow(config *config, lb *client.Client) (*workflow, error) {
	wf := &workflow{
		config: config,
		lb:     lb,
	}
	if err := wf.buildActions(); err != nil {
		return nil, err
	}
	return wf, nil
}

// run runs our workflow on the supplied "actions" doing "canaryNum" canaries,
// then running "concurrency" number of actions that will stop at "maxFailures" number of
// failurea.
func (w *workflow) run(ctx context.Context) error {
	// Run a local precondition to make sure our load balancer is in a healthy state.
	preCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	if err := w.checkLBState(preCtx); err != nil {
		w.endState = esPreconditionFailure
		return fmt.Errorf("checkLBState precondition fail: %s", err)
	}
	cancel()

	// Run our canaries one at a time. Any problem stops the workflow.
	for i := 0; i < len(w.actions) && int32(i) < w.config.CanaryNum; i++ {
		color.Green("Running canary on: %s", w.actions[i].endpoint)
		ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		err := w.actions[i].run(ctx)
		cancel()
		if err != nil {
			w.endState = esCanaryFailure
			return fmt.Errorf("canary failure on endpoint(%s): %w\n", w.actions[i].endpoint, err)
		}
		color.Yellow("Sleeping after canary for 1 minutes")
		time.Sleep(1 * time.Minute)
	}

	limit := make(chan struct{}, w.config.Concurrency)
	wg := sync.WaitGroup{}

	// Run the rest of the actions, with a limit to our concurrency.
	for i := w.config.CanaryNum; int(i) < len(w.actions); i++ {
		i := i
		limit <- struct{}{}
		if atomic.LoadInt32(&w.failures) > w.config.MaxFailures {
			break
		}
		wg.Add(1)
		go func() {
			defer func() { <-limit }()
			defer wg.Done()
			ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)

			color.Green("Upgrading endpoint: %s", w.actions[i].endpoint)
			err := w.actions[i].run(ctx)
			cancel()
			if err != nil {
				color.Red("Endpoint(%s) had upgrade error: %s", w.actions[i].endpoint, err)
				atomic.AddInt32(&w.failures, 1)
			}
		}()
	}
	wg.Wait()

	if atomic.LoadInt32(&w.failures) > w.config.MaxFailures {
		w.endState = esMaxFailures
		return errors.New("exceeded max failures")
	}
	w.endState = esSuccess
	return nil
}

// retryFailed retries all failed actiona. This is only used if
func (w *workflow) retryFailed(ctx context.Context) {
	if w.endState != esSuccess {
		panic("retrlyFailed cannot be called unless the workflow was a success")
	}

	ws := w.status()

	wg := sync.WaitGroup{}

	for i := 0; i < len(ws.failures); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)

			err := ws.failures[i].run(ctx)
			cancel()
			if err == nil {
				atomic.AddInt32(&w.failures, -1)
			}
		}()
	}
	wg.Wait()
}

// checkLBState checks the load balancer pool for "pattern" contains all "endpoints"
// in a healthy state.
func (w *workflow) checkLBState(ctx context.Context) error {
	ph, err := w.lb.PoolHealth(ctx, w.config.Pattern, true, true)
	if err != nil {
		return fmt.Errorf("PoolHealth(%s) error: %w", w.config.Pattern, err)
	}
	switch ph.Status {
	case pb.PoolStatus_PS_EMPTY:
	case pb.PoolStatus_PS_FULL:
		if len(w.config.Backends) != len(ph.Backends) {
			return fmt.Errorf("expected backends(%d) != found backends(%d)", len(w.config.Backends), len(ph.Backends))
		}
		m := map[string]bool{}
		for _, e := range w.config.Backends {
			m[e] = true
		}
		for _, hb := range ph.Backends {
			switch {
			case hb.Backend.GetIpBackend() != nil:
				b := hb.Backend.GetIpBackend()
				if !m[b.Ip] {
					return fmt.Errorf("configured backend %q not in config file", b.Ip)
				}
			default:
				return fmt.Errorf("we only support IPBackend, got %T", hb.Backend)
			}
		}
	default:
		return fmt.Errorf("pool was not at full health, was %s", ph.Status)
	}
	return nil
}

// buildActions builds actions from our configuration file.
func (w *workflow) buildActions() error {
	for _, b := range w.config.Backends {
		a, err := newServerActions(b, w.config, w.lb)
		if err != nil {
			return err
		}
		w.actions = append(w.actions, a)
	}
	return nil
}

type workflowStatus struct {
	// endState is the endState of the workflow.
	endState endState
	// failures is a list of failed actiona.
	failures []*actions
}

// status will return the workflow's status after run() has complete.d
func (w *workflow) status() workflowStatus {
	ws := workflowStatus{endState: w.endState}
	for _, action := range w.actions {
		if action.err != nil {
			ws.failures = append(ws.failures, action)
		}
	}
	return ws
}

type stateFn func(ctx context.Context) (stateFn, error)

type actions struct {
	endpoint  string
	backend   client.IPBackend
	config    *config
	srcf      *os.File
	dst       string
	lb        *client.Client
	sshClient *ssh.Client

	started     bool
	failedState stateFn
	err         error
}

func newServerActions(endpoint string, config *config, lb *client.Client) (*actions, error) {
	ip, err := checkIP(endpoint)
	if err != nil {
		return nil, err
	}
	return &actions{
		endpoint: endpoint,
		backend:  client.IPBackend{IP: ip, Port: int32(config.BinaryPort)},
		config:   config,
		lb:       lb,
	}, nil
}

func (a *actions) run(ctx context.Context) (err error) {
	a.srcf, err = os.Open(a.config.Src)
	if err != nil {
		a.err = fmt.Errorf("cannot open binary to copy(%s): %w", a.config.Src, err)
		return a.err
	}

	back := a.endpoint + ":22"
	a.sshClient, err = ssh.Dial("tcp", back, a.config.ssh)
	if err != nil {
		a.err = fmt.Errorf("problem dialing the endpoint(%s): %w", back, err)
		return a.err
	}
	defer a.sshClient.Close()

	fn := a.rmBackend
	if a.failedState != nil {
		fn = a.failedState
	}

	a.started = true
	for {
		if ctx.Err() != nil {
			a.err = ctx.Err()
			return ctx.Err()
		}
		fn, err = fn(ctx)
		if err != nil {
			a.failedState = fn
			a.err = err
			return err
		}
		if fn == nil {
			return nil
		}
	}
}

func (a *actions) rmBackend(ctx context.Context) (stateFn, error) {
	err := a.lb.RemoveBackend(ctx, a.config.Pattern, a.backend)
	if err != nil {
		return nil, fmt.Errorf("problem removing backend from pool: %w", err)
	}

	return a.jobKill, nil
}

func (a *actions) jobKill(ctx context.Context) (stateFn, error) {
	pids, err := a.findPIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("problem finding existing PIDs: %w", err)
	}

	if len(pids) == 0 {
		return a.cp, nil
	}

	if err := a.killPIDs(ctx, pids, 15); err != nil {
		return nil, fmt.Errorf("failed to kill existing PIDs: %w", err)
	}

	if err := a.waitForDeath(ctx, pids, 30*time.Second); err != nil {
		if err := a.killPIDs(ctx, pids, 9); err != nil {
			return nil, fmt.Errorf("failed to kill existing PIDs: %w", err)
		}
		if err := a.waitForDeath(ctx, pids, 10*time.Second); err != nil {
			return nil, fmt.Errorf("failed to kill existing PIDs after -9: %w", err)
		}
		return a.cp, nil
	}
	return a.cp, nil
}

func (a *actions) cp(ctx context.Context) (stateFn, error) {
	if err := a.sftp(); err != nil {
		return nil, fmt.Errorf("failed to cp binary to remote end: %w", err)
	}
	return a.jobStart, nil
}

func (a *actions) jobStart(ctx context.Context) (stateFn, error) {
	if err := a.runBinary(ctx); err != nil {
		return nil, fmt.Errorf("failed to start binary after copy: %w", err)
	}
	return a.reachable(ctx)
}

func (a *actions) reachable(ctx context.Context) (stateFn, error) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	c := &http.Client{}

	u := &url.URL{
		Host:   net.JoinHostPort(a.endpoint, strconv.Itoa(a.config.BinaryPort)),
		Path:   "/healthz",
		Scheme: "http",
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		u.String(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("problem creating HTTP request: %w", err)
	}

	for {
		if ctx.Err() != nil {
			return nil, errors.New("reachable() timed out")
		}

		resp, err := c.Do(req)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		if strings.TrimSpace(string(b)) == "ok" {
			return a.addBackend, nil
		}
	}
}

func (a *actions) addBackend(ctx context.Context) (stateFn, error) {
	err := a.lb.AddBackend(ctx, a.config.Pattern, a.backend)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (a *actions) findPIDs(ctx context.Context) ([]string, error) {
	serviceName := path.Base(a.config.Src)

	result, err := a.combinedOutput(
		ctx,
		a.sshClient,
		fmt.Sprintf("pidof %s", serviceName),
	)
	if err != nil {
		if err.(*ssh.ExitError).ExitStatus() == 127 {
			return nil, err
		}
		return nil, nil
	}

	return strings.Split(strings.TrimSpace(result), " "), nil
}

func (a *actions) killPIDs(ctx context.Context, pids []string, signal syscall.Signal) error {
	for _, pid := range pids {
		_, err := a.combinedOutput(
			ctx,
			a.sshClient,
			fmt.Sprintf("kill -s %d %s", signal, pid),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *actions) waitForDeath(ctx context.Context, pids []string, timeout time.Duration) error {
	t := time.NewTimer(timeout)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			return errors.New("timeout waiting for pids death")
		default:
		}

		results, err := a.findPIDs(ctx)
		if err != nil {
			return fmt.Errorf("findPIDs giving errors: %w", err)
		}

		if len(results) == 0 {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
}

func (a *actions) runBinary(ctx context.Context) error {
	err := a.startOnly(
		ctx,
		a.sshClient,
		fmt.Sprintf("/usr/bin/nohup %s &", a.config.Dst),
	)
	if err != nil {
		return fmt.Errorf("problem running the binary on the remove side: %w", err)
	}
	return nil
}

func (a *actions) sftp() error {
	c, err := sftp.NewClient(a.sshClient)
	if err != nil {
		return fmt.Errorf("could not create SFTP client: %w", err)
	}
	defer c.Close()

	dstf, err := c.OpenFile(a.config.Dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return fmt.Errorf("SFTP could not open file on remote destination(%s): %w", a.config.Dst, err)
	}
	defer dstf.Close()
	if err := dstf.Chmod(0770); err != nil {
		return fmt.Errorf("SFTP could not set the file mode to 0770: %w", err)
	}

	_, err = io.Copy(dstf, a.srcf)
	if err != nil {
		return fmt.Errorf("SFTP failed to do a complete copy: %w", err)
	}
	return nil
}

// combinedOutput runs a command on an SSH client. The context can be cancelled, however
// SSH does not always honor the kill signals we send, so this might not break. So closing
// the session does nothing. So depending on what the server is doing, cancelling the context
// may do nothing and it may still block.
func (*actions) combinedOutput(ctx context.Context, conn *ssh.Client, cmd string) (string, error) {
	sess, err := conn.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()

	if v, ok := ctx.Deadline(); ok {
		t := time.NewTimer(v.Sub(time.Now()))
		defer t.Stop()

		go func() {
			x := <-t.C
			if !x.IsZero() {
				sess.Signal(ssh.SIGKILL)
			}
		}()
	}

	b, err := sess.Output(cmd)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (*actions) startOnly(ctx context.Context, conn *ssh.Client, cmd string) error {
	sess, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("could not start new SSH session: %w", err)
	}
	// Note: don't close the session, it will prevent the program from starting.

	return sess.Start(cmd)
}

func (a *actions) failure() string {
	if a.failedState == nil {
		return ""
	}

	return runtime.FuncForPC(reflect.ValueOf(a.failedState).Pointer()).Name()
}
