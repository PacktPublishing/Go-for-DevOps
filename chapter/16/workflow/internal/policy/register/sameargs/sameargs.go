/*
Package sameargs defines a generic policy that can be used to look at Jobs of certain types and
validate that every Job of that type has certain arguments that are the same for every invocation.

This can be used, for example, to restrict something to working on one service, router, site, region, ...
*/
package sameargs

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
	policy.Register("sameArgs", p, Settings{})
}

// ArgKeys are a list of string
type ArgKeys []string

// Settings provides settings for a specific implementation of our Policy.
type Settings struct {
	// Jobs are a list of Job names we want to check and the args
	// that must be the same across each Job with that name.
	Jobs map[string]ArgKeys
}

// Settings implements policy.Settings.Validate().
func (s Settings) Validate() error {
	for name := range s.Jobs {
		if _, err := jobs.GetJob(name); err != nil {
			return fmt.Errorf("Job(%s) was not found", name)
		}
	}
	return nil
}

// checkJob returns true if we have a setting corresponding to the JobType.
func (s Settings) checkJob(name string) bool {
	_, ok := s.Jobs[name]
	return ok
}

// needKey simply looks at our Jobs argument and determines if we care about
// a specific arg key for a JobType.
func (s Settings) needKey(name string, k string) bool {
	keys, ok := s.Jobs[name]
	if !ok {
		return false
	}
	for _, key := range keys {
		if k == key {
			return true
		}
	}
	return false
}

// sameCheck holds a mapping of JobType that holds args we care about and
// the value that should be the same through every instance.
type sameCheck map[string]map[string]string

// isSame checks that a key for a JobType has the same value as "v". If
// a value hasn't been stored, it is stored and used on every future check.
func (s sameCheck) isSame(name, k, v string) bool {
	kv, ok := s[name]
	if !ok {
		s[name] = map[string]string{k: v}
		return true
	}

	stored, ok := kv[k]
	if !ok {
		s[name][k] = v
		return true
	}
	if stored == v {
		return true
	}
	return false
}

// Policy implements policy.Policy.
type Policy struct{}

// New is the constructor for Polixy.
func New() (Policy, error) {
	return Policy{}, nil
}

// Run implements Policy.Run().
func (p Policy) Run(ctx context.Context, req *pb.WorkReq, settings policy.Settings) error {
	s, ok := settings.(Settings)
	if !ok {
		return fmt.Errorf("settings were not valid type, were %T", settings)
	}

	same := sameCheck{}

	for blockNum, block := range req.Blocks {
		for jobNum, job := range block.Jobs {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if !s.checkJob(job.Name) {
				continue
			}
			if err := p.argSame(s, job, same, blockNum, jobNum); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p Policy) argSame(settings Settings, job *pb.Job, same sameCheck, blockNum, jobNum int) error {
	const policyErrMsg = "block(%d)/job(%d) violated rule: setting(%s) is different for this job"

	for k, v := range job.Args {
		if settings.needKey(job.Name, k) { // Only check if we care about the key
			if !same.isSame(job.Name, k, v) {
				return fmt.Errorf(policyErrMsg, blockNum, jobNum, k)
			}

		}
	}
	return nil
}
