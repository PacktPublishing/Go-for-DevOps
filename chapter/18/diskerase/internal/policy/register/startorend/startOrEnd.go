/*
Package startorend implements a policy that can be used to check that a WorkReq
has a particular Job in the first block or the last block with certain settings. Inaddition it can allow for certain other jobs to be before of after it (depending on other settings).

This is useful when you need certain cleanup Jobs, health checks or init jobs to be present.
*/
package startorend

import (
	"context"
	"fmt"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/18/diskerase/internal/policy"
	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/18/diskerase/proto"
)

// This registers our policy with the service.
func init() {
	p, err := New()
	if err != nil {
		panic(err)
	}
	policy.Register("startOrEnd", p)
}

// Settings provides settings for a specific implementation of our Policy.
type Settings struct {
	// JobType is the type of job that must be present.
	JobType string
	// MustArgs indicate arguments that must be set to a certain setting.
	MustArgs map[string]string
	// Start indicates this must be one of the first jobs.
	Start bool
	// End indicates this must be one of the end jobs.
	End bool
	// AllowedBeforeOrAfter are job types that are allowed before
	// this job if Start == true, or after this job if End == true.
	AllowedBeforeOrAfter []string

	jt pb.JobType
	allowed map[pb.JobType]bool
}

func (s Settings) compile(name string) (Settings, error){
	s.allowed = map[pb.JobType]bool{}
	jt, ok := pb.JobType_value[s.JobType]
	if !ok {
		return Settings{}, fmt.Errorf("policy(%s): JobType(%s) is invalid", name, s.JobType)
	}
	s.jt = pb.JobType(jt)
	if s.Start && s.End {
		return Settings{}, fmt.Errorf("policy(%s): Start and End cannot both be true", name)
	}
	if !s.Start && !s.End {
		return Settings{}, fmt.Errorf("policy(%s): either Start of End must be set", name)
	}

	for _, jts := range s.AllowedBeforeOrAfter {
		jt, ok := pb.JobType_value[jts]
		if !ok {
			return Settings{}, fmt.Errorf("policy(%s): AllowedBeforeOrAfter had JobType(%s) that is invalid")
		}
		s.allowed[pb.JobType(jt)] = true
	}
	return s, nil
}

// Policy implements policy.Policy.
type Policy struct{
}

// New is the constructor for Policy.
func New() (Policy, error) {
	return Policy{}, nil
}

// Run implements Policy.Run().
func (p Policy) Run(ctx context.Context, name string, req *pb.WorkReq, settings interface{}) error {
	s, ok := settings.(Settings)
	if !ok {
		return fmt.Errorf("settings were not valid")
	}
	var err error
	s, err = s.compile(name)
	if err != nil {
		return err
	}

	return p.eachWorkReq(ctx, name, req, s)
}

func (p Policy) eachWorkReq(ctx context.Context, name string, req *pb.WorkReq, s Settings) error {
	if s.Start {
		err := p.startOfBlock(ctx, req.Blocks[0].Jobs, s)
		if err != nil {
			return fmt.Errorf("policy(%s): requires JobType(%s) in the first block: %s", name, s.JobType, err)
		}
		return err
	}

	err := p.endOfBlock(ctx, req.Blocks[len(req.Blocks)-1].Jobs, s)
	if err != nil {
		err = fmt.Errorf("policy(%s): requires JobType(%s) in the last block: %s", name, s.JobType, err)
		return err
	}
	return nil
}

func (p Policy) startOfBlock(ctx context.Context, block []*pb.Job, s Settings) error {
	for _, job := range block {
		if job.Type == s.jt {
			return p.mustHave(ctx, job, s)
		}
		if s.allowed[job.Type] {
			continue
		}
		return fmt.Errorf("not found at the beginning of the block")
	}
	return fmt.Errorf("not found in the block at all")
}

func (p Policy) endOfBlock(ctx context.Context, block []*pb.Job, s Settings) error {
	for i := len(block); i > 0; i-- {
		job := block[i]
		if job.Type == s.jt {
			return p.mustHave(ctx, job, s)
		}
		if s.allowed[job.Type] {
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
