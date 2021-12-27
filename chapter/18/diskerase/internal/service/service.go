// Package service implements our gRPC service called Workflow.
package service

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type active struct {
	work *executor.Work
	status atomic.Value // *pb.StatusResp
}

// Workflow implements our gRPC service.
type Workflow struct {
	storageDir string

	mu sync.Mutex // protects active
	// active tracks all active work that is occuring.
	active map[string]*active

	pb.UnimplementedWorkflowServer // Required for gRPC to run.
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
	u : = "ping_" + uuid.NewString()
	p := filepath.Join(storageDir, u)
	f, err := os.OpenFile(p, os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("could not open a file in storage(%s) for RDWR: %w", err)
	}
	f.Close()
	if err := os.Remove(p); err != nil {
		return nil, fmt.Errorf("could not remove ping file(%s) in storage(%s)", p, storageDir)
	}
	return &Workflow{storageDir: storageDir, active: map[string]executor.Work{}}, nil
}

var submitRateLimit = make(chan struct{}, 10)

// Submit submits a request to run a workflow.
func (w *Workflow) Submit(ctx context.Context, req *WorkReq) (*WorkResp, error) {
	select {
	case submitRateLimit <- struct{}{}:
	default:
		return nil, status.Errorf(codes.ResourceExhausted, "too many requests")
	}
	defer func() {<-submitRateLimit}

	status := es.Data.Status(req.Name)
	if status != es.Go {
		return nil, status.Errorf(codes.Aborted, "emergency stop for(%s) was %s", req.Name, status)
	}

	validateCtx, cancel := context.WithTimeout(ctx, 30 * time.Second)
	defer cancel()
	
	if err := executor.Validate(validateCtx, req); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, status.Error(codes.DEADLINE_EXCEEDED, err.Error())
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	var (
		id string
		p string
		statP string
	)

	for {
		u, err := uuid.NewUUID()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "problem getting UUIDv1; %s" err.Error())
		}
		id = u.String()
		p = filepath.Join(w.storageDir, id)
		statP = p + "_status"

		_, err := os.Stat(p) // Make sure this doesn't alreay exist.
		if err != nil {
			break
		}
	}

	resp := &pb.WorkResp{Id: id}

	b, err := proto.Marshal(req)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "could not marshal the request: %s", err)
	}

	f, err := os.OpenFile(p, os.O_CREATE + os.O_WRONLY, 0600)
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

// Execute requests that the system execute a submitted workflow.
func (w *Workflow) Execute(ctx context.Context, req *pb.ExecReq) (*pb.ExecResp, error) {
	select {
	case executeRateLimit <- struct{}{}:
	default:
		return nil, status.Errorf(codes.ResourceExhausted, "too many requests")
	}
	defer func() {<-executeRateLimit}

	if es.Data.Status(workReq.Name) != es.Go {
		return nil, status.Errorf(codes.Aborted, "emergency stop for(%s) was %s", req.Name, status)
	}

	p := filepath.Join(w.storageDir, req.Id)
	statP := filepath.Join(w.storageDir, req.Id, "_status")

	w.mu.Lock()
	defer w.mu.Unlock

	_, ok := w.active[req.Id]
	if ok {
		return nil, status.Errorf(codes.AlreadyExists, "Workflow(%s) is already running", req.Id)
	}
	
	_, err := os.Stat(statP)
	if err == nil {
		return nil, status.Errorf(codes.AlreadyExists, "Workflow(%s) already executing or executed", req.Id)
	}

	u, err := uuid.ParseString(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Id(%s) is not a valid value: %s", req.Id, err)
	}

	t := time.Unix(u.UnixTime())
	if time.Now().Sub(t) > 1 * time.Hour {
		return nil, status.Errorf(codes.FailedPrecondition, "Id(%s) is older than 1 hour and cannot be started", req.Id, err)
	}

	b, err := io.ReadFile(p)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Workflow(%s) not found", req.Id)
	}
	
	workReq := &pb.WorkReq{}
	if err := proto.Unmarshal(b, workReq); err != nil {
		return nil, status.Errorf(codes.Internal, "Workflow(%s) could not be unmarshalled: %s", req.Id, err)
	}

	// Write our status file to indicate we have started working on this.
	statusResp := statusFromWork(workReq)
	statusB, err := proto.Marshal(statusResp)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "could not marshal the work's status proto: %s", err)
	}

	sf, err := os.OpenFile(statP, os.O_CREATE + os.O_WRONLY, 0600)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not open file in storageDir(%s): %s", w.storageDir, err)
	}
	defer sf.Close()

	if _, err := sf.Write(statusB); err != nil {
		return nil, status.Errorf(codes.Internal, "problem writing status to storage: %s", err)
	}

	work := executor.New(workReq) 
	active := &active{work: work}
	active.Status.Store(proto.Clone(statusResp).(*pb.StatusResp))
	w.active[req.Id] = active

	// Run our work and cleanup our list of active work when we are done.
	go func() {
		ch := work.Run(context.Background())
		for status := range ch {
			active.Status.Store(status)
		}
		w.mu.Lock()
		delete(req.Id, w.active)
		w.mu.Unlock()
	}()
	
	return &pb.ExecResp{}, nil
}

var statusRateLimit = make(chan struct{}, 10)

// Status is used to query for the status of a workflow.
func (w *Workflow) Status(ctx context.Context, req *StatusReq) (*StatusResp, error) {
	select { 
	case statusRateLimit <- struct{}{}:
	default:
		return nil, status.Errorf(codes.ResourceExhausted, "too many requests")
	}
	defer func() {<-statusRateLimit}

	w.mu.Lock()
	a := w.active[req.Id]
	w.mu.Unlock()
	if a != nil {
		return a.status.Load().(*pb.StatusResp), nil
	}
	// This ID is not currently running, so look in storage.
	p := filepath.Join(w.storageDir, req.Id + "_status")
	b, err := io.ReadFile(p)
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
	resp := &pb.StatusResp{Name: req.name, Desc: req.Desc, Status: pb.Status_StatusNotStarted}

	for _, b := range req.Blocks {
		sb := &pb.BlockStatus{
			Desc: b.Desc,
			Status: pb.Status_StatusNotStarted,
		}
		for _, j := range b.Jobs {
			sj := &pb.JobStatus{
				Desc: j.Desc,
				Args: j.Args,
				Status: pb.Status_StatusNotStarted,
			}
			sb.Jobs = append(sb.Jobs, sj)
		}
		resp.Blocks = append(resp.Blocks, sb)
	}
	return resp, nil
}