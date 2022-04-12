/*
Package validatedecom registers a job that is used to validate a site is set to state "decom".

Register name: "validateDecom"
Args:
	"site"(mandatory): The name of the site, like "aaa" or "aba"
	"siteType"(mandatory): The type of the site, like "satellite" or "cluster"
Result:
	If the site is not in decom, will return a fatal error.
*/
package sleep

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/data/packages/sites"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/service/jobs"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/proto"
)

// This registers our Job on server startup.
func init() {
	jobs.Register("sleep", newJob())
}

type args struct {
	d time.Duration
}

func (a *args) validate(args map[string]string) error {
	must := map[string]bool{
		"seconds": false,
	}

	for k, v := range args {
		switch k {
		case "seconds":
			i, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("arg(seconds) is not an integer(%s)", v)
			}
			if i < 1 {
				return fmt.Errorf("arg(seconds) cannot be less than 1(%d)", i)
			}
			must["seconds"] = true
			a.d = time.Duration(i) * time.Second
		default:
			return fmt.Errorf("validateDecom had invalid arg(%s)", k)
		}
	}

	for k, v := range must {
		if !v {
			return fmt.Errorf("missing required arg(%s)", k)
		}
	}
	return nil
}

// Job implements jobs.Job.
type Job struct {
	sites map[string]sites.Site
	args  args
}

func newJob() *Job {
	return &Job{}
}

// Validate implements jobs.Job.Validate().
func (j *Job) Validate(job *pb.Job) error {
	a := args{}
	if err := a.validate(job.Args); err != nil {
		return err
	}
	j.args = a
	return nil
}

// Run implements jobs.Job.Run().
func (j *Job) Run(ctx context.Context, job *pb.Job) error {
	time.Sleep(j.args.d)
	return nil
}
