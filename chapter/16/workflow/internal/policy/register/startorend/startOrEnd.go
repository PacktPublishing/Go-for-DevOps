/*
Package startorend implements a policy that can be used to check that a WorkReq
has a particular Job in the first block or the last block with certain settings. Inaddition it can allow for certain other jobs to be before of after it (depending on other settings).

This is useful when you need certain cleanup Jobs, health checks or init jobs to be present.
*/
package startorend

import (
	"context"
	"fmt"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/policy"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/service/jobs"
	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/proto"
)

// This registers our policy with the service.
func init() {
	p, err := New()
	if err != nil {
		panic(err)
	}
	policy.Register("startOrEnd", p, Settings{})
}

// Settings provides settings for a specific implementation of our Policy.
type Settings struct {
	// JobName is the name of the job that must be present.
	JobName string
	// MustArgs indicate arguments that must be set to a certain setting.
	MustArgs map[string]string
	// Start indicates this must be one of the first jobs.
	Start bool
	// End indicates this must be one of the end jobs.
	End bool
	// AllowedBeforeOrAfter are job types that are allowed before
	// this job if Start == true, or after this job if End == true.
	AllowedBeforeOrAfter []string

	allowed map[string]bool
}

func (s Settings) Validate() error {
	if _, err := jobs.GetJob(s.JobName); err != nil {
		return fmt.Errorf("Job(%s) is invalid", s.JobName)
	}

	if s.Start && s.End {
		return fmt.Errorf("Start and End cannot both be true")
	}
	if !s.Start && !s.End {
		return fmt.Errorf("either Start of End must be set")
	}

	for _, name := range s.AllowedBeforeOrAfter {
		if _, err := jobs.GetJob(name); err != nil {
			return fmt.Errorf("AllowedBeforeOrAfter had Job(%s) that is invalid", name)
		}
	}
	return nil
}

func (s Settings) compile() Settings {
	s.allowed = map[string]bool{}
	for _, name := range s.AllowedBeforeOrAfter {
		s.allowed[name] = true
	}
	return s
}

// Policy implements policy.Policy.
type Policy struct {
}

// New is the constructor for Policy.
func New() (Policy, error) {
	return Policy{}, nil
}

// Run implements Policy.Run().
func (p Policy) Run(ctx context.Context, req *pb.WorkReq, settings policy.Settings) error {
	s, ok := settings.(Settings)
	if !ok {
		return fmt.Errorf("settings were not valid type, were %T", settings)
	}

	return p.eachWorkReq(ctx, req, s.compile())
}

func (p Policy) eachWorkReq(ctx context.Context, req *pb.WorkReq, s Settings) error {
	if s.Start {
		err := p.startOfBlock(ctx, req.Blocks[0].Jobs, s)
		if err != nil {
			return fmt.Errorf("requires Job(%s) in the first block: %s", s.JobName, err)
		}
		return err
	}

	err := p.endOfBlock(ctx, req.Blocks[len(req.Blocks)-1].Jobs, s)
	if err != nil {
		err = fmt.Errorf("requires Job(%s) in the last block: %s", s.JobName, err)
		return err
	}
	return nil
}

func (p Policy) startOfBlock(ctx context.Context, block []*pb.Job, s Settings) error {
	for _, job := range block {
		if job.Name == s.JobName {
			return p.mustHave(ctx, job, s)
		}
		if s.allowed[job.Name] {
			continue
		}
		return fmt.Errorf("not found at the beginning of the block")
	}
	return fmt.Errorf("not found in the block at all")
}

func (p Policy) endOfBlock(ctx context.Context, block []*pb.Job, s Settings) error {
	for i := len(block); i > 0; i-- {
		job := block[i]
		if job.Name == s.JobName {
			return p.mustHave(ctx, job, s)
		}
		if s.allowed[job.Name] {
			continue
		}
		return fmt.Errorf("not found at the beginning of the block")
	}
	return fmt.Errorf("not found in the block at all")
}

func (p Policy) mustHave(ctx context.Context, job *pb.Job, s Settings) error {
	for k, v := range s.MustArgs {
		has, ok := job.Args[k]
		if !ok {
			return fmt.Errorf("found, but required arg(%s) was not found", k)
		}
		if v != has {
			return fmt.Errorf("found,but required  arg(%s) found has incorrect value(got %q, want %q)", k, has, v)
		}
	}
	return nil
}
