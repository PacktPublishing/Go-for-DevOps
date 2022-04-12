/*
Package diskerase registers a job that can be used to erase a disk on a machine. As
this is just a demo, this really just sleeps for 30 seconds.

Register name: "diskErase"
Args:
	"machine"(mandatory): The name of the machine, like "aa01" or "ab02"
	"site"(mandatory): The name of the site, like "aaa" or "aba"
Result:
	Erases a disk on a machine, except this is a demo, so it really just sleeps for 30 seconds.
*/
package diskerase

import (
	"context"
	"fmt"
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/data/packages/sites"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/service/jobs"
	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/proto"
)

// This registers our Job on server startup.
func init() {
	jobs.Register("diskErase", newJob())
}

type args struct {
	machine string
	site    string
}

func (a *args) validate(args map[string]string) error {
	must := map[string]bool{
		"machine": false,
		"site":    false,
	}

	for k, v := range args {
		switch k {
		case "machine":
			must["machine"] = true
			a.machine = v
		case "site":
			if _, ok := sites.Data.Sites[v]; !ok {
				return fmt.Errorf("site(%s) arg was not a valid site", v)
			}
			must["site"] = true
			a.site = v

		default:
			return fmt.Errorf("invalid arg(%s)", k)
		}
	}

	for k, v := range must {
		if !v {
			return fmt.Errorf("missing required arg(%s)", k)
		}
	}

	fullName := fmt.Sprintf("%s.%s", a.machine, a.site)
	_, ok := sites.Data.Machines[fullName]
	if !ok {
		return fmt.Errorf("invalid arg(machine): machine(%s) does not exist", fullName)
	}

	return nil
}

// Job implements jobs.Job.
type Job struct {
	args args
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
	time.Sleep(30 * time.Second) // A crude and inaccurate simulation of a disk erasure
	return nil
}
