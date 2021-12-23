/*
Package restrictjobtypes provides a policy that can be invoked to ensure that a WorkReq only contains
jobs of certain types. Any job outside these types will cause a policy violation.
*/
package restrictjobtypes

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
	policy.Register("restrictJobTypes", p)
}

// Settings provides settings for a specific implementation of our Policy.
type Settings struct {
	AllowedJobs []string

	hash map[pb.JobType]bool
}

// compile returns a copy of Settings that looks up all allowed jobs in the proto
// and creates a hash for checking if a JobType is allowed.
func (s Settings) compile() (Settings, error) {
	s.hash = map[pb.JobType]bool{}
	for _, n := range s.AllowedJobs {
		jt, ok := pb.JobType_value[n]
		if !ok {
			return s, fmt.Errorf("allowed job(%s) is not defined in the proto")
		}
		s.hash[pb.JobType(jt)] = true
	}
	return s, nil
}

// Policy implements policy.Policy.
type Policy struct{}

// New is the constructor for Policy.
func New() (Policy, error) {
	return Policy{}, nil
}

// Run implements Policy.Run().
func (p Policy) Run(ctx context.Context, name string, req *pb.WorkReq, settings interface{}) error {
	const errMsg = "policy(%s): block(%d)/job(%d) is a type(%s) that is not allowed"

	s, ok := settings.(Settings)
	if !ok {
		return fmt.Errorf("settings were not valid")
	}
	var err error
	s, err = s.compile()
	if err != nil {
		return err
	}

	for blockNum, block := range req.Blocks {
		for jobNum, job := range block.Jobs {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			if !s.hash[job.Type] {
				return fmt.Errorf(errMsg, blockNum, jobNum, name, pb.JobType_name[int32(job.Type)])
			}
		}
	}
	return nil
}
