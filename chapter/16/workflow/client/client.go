/*
Package client provides access to the workflow service. You can use this client to:
	Submit a *pb.WorkReq to the service
	Execute a *pb.WorkReq previously submitted
	Get the status of a *pb.WorkReq

See the README.md in the root workflow/ directory for more information.

SECURITY NOTICE: As this is an example for a book and is meant to be run in a secure environment, we
use grpc.WithInsecure().  Aka, not production ready.
*/
package client

import (
	"context"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/proto"
)

// Workflow represents our Workflow client. It uses a builtin circuit breaker wrapping
// an exponential backoff. The context passed to any method should be the maximum time you are
// willing to wait including all retries. If a call returns an error because the context expires,
// the circuit breaker will trip. Errors without gRPC status codes or with status codes of
// DeadlineExceeded or ResourceExhausted are considered fatal errors. Fatal errors do not
// get retries and do not trip the circuit breaker.
type Workflow struct {
	conn   *grpc.ClientConn
	client pb.WorkflowClient

	cb        *gobreaker.CircuitBreaker
	retryPool sync.Pool
}

// New creates a new Workflow instance.
func New(addr string) (*Workflow, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return &Workflow{
		conn:   conn,
		client: pb.NewWorkflowClient(conn),
		cb: gobreaker.NewCircuitBreaker(
			gobreaker.Settings{
				MaxRequests: 1,
				Interval:    20 * time.Second,
				Timeout:     30 * time.Second,
				ReadyToTrip: func(c gobreaker.Counts) bool {
					return c.ConsecutiveFailures > 1
				},
				IsSuccessful: func(err error) bool {
					if isFatal(err) {
						return true
					}
					return false
				},
			},
		),
		retryPool: sync.Pool{
			New: func() interface{} {
				return backoff.NewExponentialBackOff()
			},
		},
	}, nil
}

// Submit submits a pb.WorkReq to the server. If successful an ID will be returned that
// represents the pb.WorkReq on the server. This can be used in an Exec() call to execute
// the pb.WorkReq.
func (w *Workflow) Submit(ctx context.Context, req *pb.WorkReq) (string, error) {
	caller := func(ctx context.Context, req proto.Message) (proto.Message, error) {
		r := req.(*pb.WorkReq)
		return w.client.Submit(ctx, r)
	}
	resp, err := w.call(ctx, req, caller)
	if err != nil {
		return "", err
	}
	return resp.(*pb.WorkResp).Id, nil
}

// Exec causes the server to execute a pb.WorkReq that was previously accepted by the server
// via a Submit() call.
func (w *Workflow) Exec(ctx context.Context, id string) error {
	caller := func(ctx context.Context, req proto.Message) (proto.Message, error) {
		r := req.(*pb.ExecReq)
		return w.client.Exec(ctx, r)
	}
	_, err := w.call(ctx, &pb.ExecReq{Id: id}, caller)
	if err != nil {
		return err
	}
	return nil
}

// Status returns the status of a pb.WorkReq that was submitted to the server via the Submit()
// call.
func (w *Workflow) Status(ctx context.Context, id string) (*pb.StatusResp, error) {
	caller := func(ctx context.Context, req proto.Message) (proto.Message, error) {
		r := req.(*pb.StatusReq)
		return w.client.Status(ctx, r)
	}
	resp, err := w.call(ctx, &pb.StatusReq{Id: id}, caller)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.StatusResp), nil
}

type grpcCall = func(context.Context, proto.Message) (proto.Message, error)

// call generically calls any non-streaming gRPC endpoint that is contained within "call".
// This method will default to a timeout of 30 seconds unless Context has a deadline.
// We also have use a circuit breaker surrounding an exponential backoff on our calls. We
// only do retries when we receive a generic error (one that does not contain a gRPC status code)
// or status code: DeadlineExceeded, ResourceExhausted.
func (w *Workflow) call(ctx context.Context, req proto.Message, call grpcCall) (proto.Message, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	var resp proto.Message
	op := func() error {
		var err error
		resp, err = call(ctx, req)
		return err
	}

	retry := func() error {
		back := w.retryPool.Get().(*backoff.ExponentialBackOff)
		defer w.retryPool.Put(back)
		defer back.Reset()

		// Execute our op() func until the context is cancelled with exponential backoff.
		for {
			err := op()
			if err == nil {
				return nil
			}

			if isFatal(err) {
				return err
			}

			// We manually are going to do our backoff instead of using backoff.Retry()
			// because we want to differentiate between fatal errors that should never
			// get retried and errors that just need backing off.
			backoff := back.NextBackOff()
			deadline, _ := ctx.Deadline()
			if time.Now().Add(backoff).After(deadline) {
				// This happens when our next retry would happen after our deadline.
				return context.DeadlineExceeded
			}

			// Sleep before our next retry on whatever backoff we were given or
			// until our context is cancelled by the user.
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return err
			case <-timer.C:
			}
		}
	}

	_, err := w.cb.Execute(
		func() (interface{}, error) {
			return nil, retry()
		},
	)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func isFatal(err error) bool {
	if s, ok := status.FromError(err); ok {
		switch s.Code() {
		case codes.DeadlineExceeded, codes.ResourceExhausted:
			return false
		}
	}
	return true
}
