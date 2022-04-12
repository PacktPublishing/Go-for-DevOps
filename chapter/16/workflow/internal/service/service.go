// Package service implements our gRPC service called Workflow.
package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/es"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/service/executor"
	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/proto"
)

// active provides an entry for an actively executing workflow.
type active struct {
	work   *executor.Work
	status atomic.Value // *pb.StatusResp
}

// Workflow implements our gRPC service.
type Workflow struct {
	// storageDir is where we store workflow information.
	storageDir string

	// mu protects active
	mu sync.Mutex
	// active tracks all active work that is occuring.
	active map[string]*active

	// Required for gRPC to run, makes sure we have all the methods defined.
	pb.UnimplementedWorkflowServer
}

// New creates a new Workflow service.
func New(storageDir string) (*Workflow, error) {
	stat, err := os.Stat(storageDir)
	if err != nil {
		return nil, fmt.Errorf("could not stat the workflow storage(%s): %w", storageDir, err)
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("storageDir(%s) is not a directory", storageDir)
	}
	u := "ping_" + uuid.NewString()
	p := filepath.Join(storageDir, u)
	f, err := os.OpenFile(p, os.O_CREATE+os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("could not open a file in storage(%s) for RDWR: %w", storageDir, err)
	}
	f.Close()
	if err := os.Remove(p); err != nil {
		return nil, fmt.Errorf("could not remove ping file(%s) in storage(%s)", p, storageDir)
	}
	return &Workflow{storageDir: storageDir, active: map[string]*active{}}, nil
}

var submitRateLimit = make(chan struct{}, 10)

// Submit submits a request to run a workflow.
func (w *Workflow) Submit(ctx context.Context, req *pb.WorkReq) (*pb.WorkResp, error) {
	select {
	case submitRateLimit <- struct{}{}:
	default:
		return nil, status.Errorf(codes.ResourceExhausted, "too many requests")
	}
	defer func() { <-submitRateLimit }()

	esStatus := es.Data.Status(req.Name)
	if esStatus != es.Go {
		return nil, status.Errorf(codes.Aborted, "emergency stop for(%s) was %s", req.Name, esStatus)
	}

	validateCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := executor.Validate(validateCtx, req); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, status.Error(codes.DeadlineExceeded, err.Error())
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	var (
		id string
		p  string
	)

	// Loop until we get a unique ID that doesn't exist on the filesystem.
	for {
		u, err := uuid.NewUUID()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "problem getting UUIDv1; %s", err.Error())
		}
		id = u.String()
		p = filepath.Join(w.storageDir, id)

		_, err = os.Stat(p) // Make sure this doesn't alreay exist.
		if err != nil {
			break
		}
	}

	resp := &pb.WorkResp{Id: id}

	b, err := proto.Marshal(req)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "could not marshal the request: %s", err)
	}

	f, err := os.OpenFile(p, os.O_CREATE+os.O_WRONLY, 0600)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not open file in storageDir(%s): %s", w.storageDir, err)
	}
	defer f.Close()

	if _, err := f.Write(b); err != nil {
		return nil, status.Errorf(codes.Internal, "problem writing request to storage: %s", err)
	}

	return resp, nil
}

var executeRateLimit = make(chan struct{}, 10)

// Exec requests that the system execute a submitted workflow.
func (w *Workflow) Exec(ctx context.Context, req *pb.ExecReq) (*pb.ExecResp, error) {
	select {
	case executeRateLimit <- struct{}{}:
	default:
		return nil, status.Errorf(codes.ResourceExhausted, "too many requests")
	}
	defer func() { <-executeRateLimit }()

	p := filepath.Join(w.storageDir, req.Id)
	statP := filepath.Join(w.storageDir, req.Id+"_status")

	w.mu.Lock()
	defer w.mu.Unlock()

	_, ok := w.active[req.Id]
	if ok {
		return nil, status.Errorf(codes.AlreadyExists, "Workflow(%s) is already running", req.Id)
	}

	_, err := os.Stat(statP)
	if err == nil {
		return nil, status.Errorf(codes.AlreadyExists, "Workflow(%s) already executing or executed", req.Id)
	}

	u, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Id(%s) is not a valid value: %s", req.Id, err)
	}

	t := time.Unix(u.Time().UnixTime())
	if time.Now().Sub(t) > 1*time.Hour {
		return nil, status.Errorf(codes.FailedPrecondition, "Id(%s) is older than 1 hour and cannot be started", req.Id)
	}

	b, err := os.ReadFile(p)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Workflow(%s) not found", req.Id)
	}

	workReq := &pb.WorkReq{}
	if err := proto.Unmarshal(b, workReq); err != nil {
		return nil, status.Errorf(codes.Internal, "Workflow(%s) could not be unmarshalled: %s", req.Id, err)
	}

	esStatus := es.Data.Status(workReq.Name)
	if esStatus != es.Go {
		return nil, status.Errorf(codes.Aborted, "emergency stop for(%s) was %s", workReq.Name, esStatus)
	}

	// Write our status file to indicate we have started working on this.
	statusResp := statusFromWork(workReq)
	statusB, err := proto.Marshal(statusResp)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "could not marshal the work's status proto: %s", err)
	}

	if err := os.WriteFile(statP, statusB, 0600); err != nil {
		return nil, status.Errorf(codes.Internal, "problem writing status to storage: %s", err)
	}

	work := executor.New(workReq, statusResp)
	active := &active{work: work}
	active.status.Store(proto.Clone(statusResp).(*pb.StatusResp))
	w.active[req.Id] = active

	// Run our work and get the first state change.
	ch := work.Run(context.Background())
	active.status.Store(<-ch)
	writeIn := statusWriter(statP)

	// Update our status as it changes in memory and on disk.
	// Cleanup our list of active work when we are done.
	go func() {
		for status := range ch {
			// Record our status in memory
			active.status.Store(status)

			// Record our status on disk. If there is an entry pending,
			// remove it for the latest entry.
			select {
			case writeIn <- status:
			default:
				select {
				case <-writeIn:
				default:
				}
				writeIn <- status
			}
		}
		w.mu.Lock()
		delete(w.active, req.Id)
		w.mu.Unlock()
	}()

	return &pb.ExecResp{}, nil
}

var statusRateLimit = make(chan struct{}, 10)

// Status is used to query for the status of a workflow.
func (w *Workflow) Status(ctx context.Context, req *pb.StatusReq) (*pb.StatusResp, error) {
	select {
	case statusRateLimit <- struct{}{}:
	default:
		return nil, status.Errorf(codes.ResourceExhausted, "too many requests")
	}
	defer func() { <-statusRateLimit }()

	w.mu.Lock()
	a := w.active[req.Id]
	w.mu.Unlock()
	if a != nil {
		return a.status.Load().(*pb.StatusResp), nil
	}
	// This ID is not currently running, so look in storage.
	p := filepath.Join(w.storageDir, req.Id+"_status")
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "work ID(%s) was not found", req.Id)
	}
	resp := &pb.StatusResp{}
	if err := proto.Unmarshal(b, resp); err != nil {
		return nil, status.Errorf(codes.Internal, "work ID(%s) data was corrupted on disk", req.Id)
	}
	return resp, nil
}

// statusFromWork takes a WorkReq and generates the corresponding StatusResp.
func statusFromWork(req *pb.WorkReq) *pb.StatusResp {
	resp := &pb.StatusResp{Name: req.Name, Desc: req.Desc, Status: pb.Status_StatusNotStarted}

	for _, b := range req.Blocks {
		sb := &pb.BlockStatus{
			Desc:   b.Desc,
			Status: pb.Status_StatusNotStarted,
		}
		for _, j := range b.Jobs {
			sj := &pb.JobStatus{
				Name:   j.Name,
				Desc:   j.Desc,
				Args:   j.Args,
				Status: pb.Status_StatusNotStarted,
			}
			sb.Jobs = append(sb.Jobs, sj)
		}
		resp.Blocks = append(resp.Blocks, sb)
	}
	return resp
}

func statusWriter(p string) (in chan *pb.StatusResp) {
	in = make(chan *pb.StatusResp, 1)

	go func() {
		for status := range in {
			b, err := proto.Marshal(status)
			if err != nil {
				log.Println("could not marshal a status proto: ", err)
				continue
			}
			if err := os.WriteFile(p, b, 0600); err != nil {
				log.Println("cannot write a status update to disk, this is bad: ", err)
				continue
			}
		}
	}()
	return in
}
