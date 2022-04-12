/*
Package executor provides the Work type which is used to execute a pb.WorkReq.
This package is the meat of the engine.

To create a Work object, simply:
	work := executor.New(req, status}

After creating a Work object, validate it:
	if err := work.Validate(); err !=nil {
		// Do something
	}

To run the Work object, do:
	ch := work.Run()

Once Run() returns, the pb.Status object passed will contain the results of running the WorkReq.
*/
package executor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/es"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/policy"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/policy/config"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/service/jobs"
	"google.golang.org/protobuf/proto"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/proto"
)

// Work is an executor for executing a WorkReq received by the server.
type Work struct {
	req *pb.WorkReq

	mu     sync.Mutex
	status *pb.StatusResp
	ch     chan *pb.StatusResp
}

// New is the constructor for Work.
func New(req *pb.WorkReq, status *pb.StatusResp) *Work {
	return &Work{
		req:    req,
		status: status,
		ch:     make(chan *pb.StatusResp, 1),
	}
}

// Run validates that a WorkReq is correct and passed policy, then executes it.
func (w *Work) Run(ctx context.Context) chan *pb.StatusResp {
	w.setWorkStatus(pb.Status_StatusRunning, false)

	go func() {
		defer close(w.ch)

		esCh, cancelES := es.Data.Subscribe(w.req.Name)
		defer cancelES()

		if <-esCh != es.Go {
			w.setWorkStatus(pb.Status_StatusFailed, true)
			return
		}

		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()

		// If we get an emergency stop, cancel our context.
		// If the context gets cancelled, then just exit.
		go func() {
			select {
			case <-ctx.Done():
				return
			case <-esCh:
				log.Println("Emergency Stop called on running workflow type ", w.req.Name)
				w.setWorkStatus(pb.Status_StatusFailed, true)
				cancel()
			}
		}()

		// Loop through each block one at a time and execute the Jobs located in them
		// at the rate limit defined for the block.
		for i, block := range w.req.Blocks {
			if ctx.Err() != nil {
				break
			}
			stat := w.status.Blocks[i]

			if err := w.runJobs(ctx, block, stat); err != nil {
				break
			}
		}

		// Record our final state based on if any of our blocks failed.
		completed := true
		for _, block := range w.status.Blocks {
			if block.Status == pb.Status_StatusFailed {
				completed = false
				w.setWorkStatus(pb.Status_StatusFailed, false)
			}
		}
		if completed {
			w.setWorkStatus(pb.Status_StatusCompleted, false)
		}
	}()

	return w.ch
}

func (w *Work) setWorkStatus(status pb.Status, esStopped bool) {
	w.mu.Lock()
	w.status.Status = status
	w.status.WasEsStopped = esStopped
	w.sendStatus(w.status)
	w.mu.Unlock()
}

func (w *Work) setBlockStatus(block *pb.BlockStatus, status pb.Status) {
	w.mu.Lock()
	block.Status = status
	w.sendStatus(w.status)
	w.mu.Unlock()
}

func (w *Work) setJobStatus(job *pb.JobStatus, status pb.Status, err string) {
	w.mu.Lock()
	job.Status = status
	job.Error = err
	w.sendStatus(w.status)
	w.mu.Unlock()
}

// sendStatus sends the status of the WorkReq on our output channel. If the channel
// is currently blocked with another status update, it removes that update for the newer one.
func (w *Work) sendStatus(status *pb.StatusResp) {
	// We clone our status to prevent any concurrent access issues once the lock around
	// sendStatus is released.
	status = proto.Clone(status).(*pb.StatusResp)
	for {
		select {
		case w.ch <- status:
			return
		default:
			select {
			case <-w.ch:
			default:
			}
		}
	}
}

func (w *Work) runJobs(ctx context.Context, block *pb.Block, blockStatus *pb.BlockStatus) error {
	ctx, cancel := context.WithCancel(ctx)

	// Setup our rate limiter.
	limit := block.RateLimit
	if limit < 1 {
		limit = 1
	}
	rateLimiter := make(chan struct{}, int(limit))

	w.setBlockStatus(blockStatus, pb.Status_StatusRunning)

	// Execute our Jobs.
	wg := sync.WaitGroup{}
	for i, job := range block.Jobs {
		i := i
		job := job

		select {
		case rateLimiter <- struct{}{}:
		case <-ctx.Done():
		}
		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-rateLimiter }()

			js := blockStatus.Jobs[i]
			j, err := jobs.GetJob(job.Name)
			if err != nil {
				cancel()
				w.setJobStatus(js, pb.Status_StatusFailed, fmt.Sprintf("a Job(%s) passed validation but when ran could not be found, bug?", job.Name))
				return
			}

			w.setJobStatus(js, pb.Status_StatusRunning, "")
			err = j.Run(ctx, job)
			if err != nil {
				if jobs.IsFatal(err) {
					cancel()
				}
				w.setJobStatus(js, pb.Status_StatusFailed, err.Error())
				return
			}

			w.setJobStatus(js, pb.Status_StatusCompleted, "")
		}()
	}

	wg.Wait()

	// If any Job failed, the block failed.
	for _, js := range blockStatus.Jobs {
		if js.Status == pb.Status_StatusFailed {
			w.setBlockStatus(blockStatus, pb.Status_StatusFailed)
			return ctx.Err()
		}
	}
	w.setBlockStatus(blockStatus, pb.Status_StatusCompleted)
	return ctx.Err()
}

// Validate validates that a WorkReq is valid. This will check that basic values are set correctly
// and run all policies for this Workflow.
func Validate(ctx context.Context, req *pb.WorkReq) error {
	for blockNum, b := range req.Blocks {
		if len(b.Jobs) == 0 {
			return fmt.Errorf("Block(%d) had 0 jobs", blockNum)
		}
		for jobNum, j := range b.Jobs {
			job, err := jobs.GetJob(j.Name)
			if err != nil {
				return fmt.Errorf("Block(%d) Job(%d) had a invalid Type(%s)", blockNum, jobNum, j.Name)
			}
			if err := job.Validate(j); err != nil {
				return fmt.Errorf("Block(%d) Job(%d)(%s) did not validate: %s)", blockNum, jobNum, j.Name, err)
			}
		}
	}

	conf, err := config.Policies.Read()
	if err != nil {
		log.Println("policy config could not be read: ", err)
		return fmt.Errorf("cannot read our policies config: %s", err)
	}
	workConf, ok := conf.Workflows[req.Name]
	if !ok {
		return fmt.Errorf("Workflow does not have an associated policy in the policy configuration file")
	}

	args := make([]policy.PolicyArgs, 0, len(workConf.Policies))
	for _, p := range workConf.Policies {
		args = append(args, policy.PolicyArgs{Name: p.Name, Settings: p.SettingsTyped})
	}

	policyContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := policy.Run(policyContext, req, args...); err != nil {
		return err
	}
	return nil
}
