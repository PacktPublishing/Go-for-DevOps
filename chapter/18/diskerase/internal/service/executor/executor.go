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
type Work struct{}

// Run validates that a WorkReq is correct and passed policy, then executes it. If we have
// no errors, this will return nil and not an empty slice.
func (w *Work) Run(ctx context.Context, req *pb.WorkReq) []error {
	if err := w.validate(ctx, req); err != nil {
		return []error{err}
	}

	errCh := make(chan error, 1)
	var errs []error

	// This collects errors that on our our concurrent jobs reports. If the error is fatal
	// it stops the other jobs.
	errDone := make(chan struct{})
	go func() {
		defer close(errDone)
		noFatal := false
		cancelled := false
		for err := range errCh {
			// We don't need to have a bunch of cancelled errors or any cancelled
			// errors if we recieved a fatal error that cancelled the jobs.
			if errors.Is(err, context.Canceled) {
				if noFatal || cancelled {
					continue
				}
				cancelled = true
			}
			errs = append(errs, err)

			if jobs.IsFatal(err) {
				noFatal = true
			}
		}
	}()

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	// Loop through each block one at a time and execute the Jobs located in them
	// at the rate limit defined for the block.
	wg := sync.WaitGroup{}
	for _, block := range req.Blocks {
		// Setup our rate limiter.
		limit := block.RateLimit
		if limit < 1 {
			limit = 1
		}
		rateLimiter := make(chan struct{}, int(limit))
		for _, job := range block.Jobs {
			rateLimiter <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-rateLimiter }()

				j, err := jobs.GetJob(job.Name)
				if err != nil {
					cancel()
					errCh <- jobs.Fatalf("a Job(%s) passed validation but when ran could not be found, bug?", job.Name)
				}
				err = j.Run(ctx, job)
				if err != nil {
					if jobs.IsFatal(err) {
						cancel()
					}
					errCh <- err
				}
			}()
		}
	}

	// Wait for our Jobs to finish executing.
	wg.Wait()
	// No more jobs to report errors, so tell our error collector we are done.
	close(errCh)
	// Wait for error collection to be done.
	<-errDone
	// Return any errors we have. If we have none, this will be nil.
	return errs
}

// Validate that a WorkReq is valid. This will check that basic values are set correctly
// and run all policies for this Workflow.
func (w *Work) Validate(ctx context.Context, req *pb.WorkReq) error {
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
