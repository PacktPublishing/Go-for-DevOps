// Package service defines our gRPC service called Workflow.
package service

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Workflow implements our gRPC service.
type Workflow struct {
	storageDir string

	mu sync.Mutex
	active map[string]*executor.Work

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

// Submit submits a request to run a workflow.
func (w *Workflow) Submit(ctx context.Context, req *WorkReq) (*WorkResp, error) {
	validateCtx, cancel := context.WithTimeout(ctx, 30 * time.Second)
	defer cancel()
	
	work := executor.Work{}
	if err := work.Validate(validateCtx, req); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, status.Error(codes.DEADLINE_EXCEEDED, err.Error())
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	var (
		id string
		submit time.Time
		p string
	)

	for {
		u, err := uuid.NewUUID()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "problem getting UUIDv1; %s" err.Error())
		}
		id = u.String()
		p = filepath.Join(w.storagedir, id)
		_, err := os.Stat(p)
		if err != nil {
			break
		}
	}

	resp := &pb.WorkResp{Id: id}

	f, err := os.OpenFile(p, os.O_CREATE + os.O_WRONLY, 0600)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not open file in storageDir(%s): %s", w.storageDir, err)
	}
	defer f.Close()

	b, err := proto.Marshal(req)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "could not marshal the request: %s", err)
	}
	if _, err := f.Write(b); err != nil {
		return nil, status.Errorf(codes.Internal, "problem writing request to storage: %s", err)
	}
	return resp, nil
}

// Execute requests that the system execute a submitted workflow.
func (w *Workflow) Execute(ctx context.Context, req *pb.ExecReq) (*pb.ExecResp, error) {
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
		return nil, status.Errorf(codes.AlreadyExists, "Workflow(%s) already ran", req.Id)
	}

	b, err := io.ReadFile(p)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Workflow(%s) not found", req.Id)
	}
	
	workReq := &pb.WorkReq{}
	if err := proto.Unmarshal(b, workReq); err != nil {
		return nil, status.Errorf(codes.Internal, "Workflow(%s) could not be unmarshalled: %s", req.Id, err)
	}

	work := &executor.Work{}
	go func() {
		errs := work.Run(context.Background(), workReq)
	}()
	active[req.Id] = work

	return &pb.ExecResp{}, nil
}

// Status is used to query for the status of a workflow.
func (w *Workflow) Status(ctx context.Context, req *StatusReq) (*StatusResp, error) {
	return nil, fmt.Errorf("unimplemented")
}
