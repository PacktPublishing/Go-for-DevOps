/*
Package executor executes all the work in the engine. 

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
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/18/diskerase/internal/policy"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/18/diskerase/internal/policy/config"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/18/diskerase/internal/service/jobs"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/18/diskerase/proto"
)

// Work is an executor for executing a WorkReq received by the server.
type Work struct{
	req *pb.WorkReq

	mu sync.Mutex
	status *pb.StatusResp
	ch chan *pb.StatusResp
}

// New is the constructor for Work.
func New(req *pb.WorkReq, status *pb.StatusResp) *Work {
	return &Work{
		req: req,
		status: status,
		ch: make(chan *pb.StatusResp),
	}
}

// Run validates that a WorkReq is correct and passed policy, then executes it.
func (w *Work) Run(ctx context.Context) chan *pb.StatusResp {
	go func() {
		defer close(w.ch)

		esCh, cancelES := es.Data.Subscribe(w.req.Name)
		defer cancelES()
		if <-esCh != es.Go {
			w.status.Status = pb.StatusFailed
			w.status.WasEsStopped = true
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
				cancel()
			}
		}()

		w.setWorkStatus(pb.Status_StatusRunning)

		// Loop through each block one at a time and execute the Jobs located in them
		// at the rate limit defined for the block.
		for _, block := range w.req.Blocks {
			if ctx.Err() != nil {
				break
			}
			
			w.runJobs(ctx, block)
		}
		
		failed = false
		for _, block := range w.status.Blocks {
			if block.Status == pb.Status_StatusFailed {
				failed = true
				w.setWorkStatus(pb.Status_StatusFailed)
			}
		}
		if !failed {
			w.setWorkStatus(pb.Status_StatusCompleted)
		}
	}()

	return ch
}

func (w *Work) setWorkStatus(status *pb.Status) {
	w.mu.Lock()
	w.status.Status = status
	w.sendStatus(w.status)
	w.mu.Unlock()
}

func (w *Work) setBlockStatus(block *pb.BlockStatus, status *pb.Status) {
	w.mu.Lock()
	block.Status = status
	w.sendStatus(w.status)
	w.mu.Unlock()
}

func (w *Work) setJobStatus(job *pb.JobStatus, status *pb.Status, err string) {
	w.mu.Lock()
	job.Status = status
	job.Error = err
	w.sendStatus(w.status)
	w.mu.Unlock()
}

// sendStatus sends the status of the WorkReq on our output channel. If the channel
// is currently blocked with another status update, it removes that update for the newer one.
func (w *Work) sendStatus(status *pb.Status) {
	// We clone our status to prevent any concurrent access issues once the lock around
	// sendStatus is released.
	status = proto.Clone(status).(*pb.Status)
	for {
		select{
		case w.ch <-status:
			return
		default:
			select{
			case <-w.ch:
			default:
			}
		}
	}
}

func (w *Work) runJobs(ctx context.Context, block *pb.Block, blockStatus *pb.BlockStatus) {
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
			return
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-rateLimiter }()

			js := blockStatus.Jobs[i]
			j, err := jobs.GetJob(job.Name)
			if err != nil {
				cancel()
				w.setJobStatus(js, pb.Status_StatusFailed,fmt.Sprintf("a Job(%s) passed validation but when ran could not be found, bug?", job.Name))
				return
			}

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
	for i, job := range block.Jobs {
		if job.Status == pb.Status_StatusFailed {
			w.setBlockStatus(blockStatus, pb.Status_StatusFailed)
			return
		}
	}
	w.setBlockStatus(blockStatus, pb.Status_StatusCompleted)
}

// Validate validates that a WorkReq is valid. This will check that basic values are set correctly
// and run all policies for this Workflow.
func Validate(ctx context.Context, req *pb.WorkReq) error {
	for blockNum, b := range req.Blocks {
		if len(b.Jobs) == 0 {
			return fmt.Errorf("Block(%d) had 0 jobs", blockNum)
		}
		for jobNum, j := range b.Jobs {
			job, err := jobs.GetJob(j.Name); err != nil {
				return fmt.Errorf("Block(%d) Job(%d) had a invalid Type(%s)", blockNum, jobNum, j.Name)
			}
			if err := job.Validate(j); err != nil {
				return fmt.Errorf("Block(%d) Job(%d) did not validate: %s)", blockNum, jobNum, err)
			}
		}
	}

	conf, err := config.Policies.Read()
	if err != nil {
		log.Println(err)
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
