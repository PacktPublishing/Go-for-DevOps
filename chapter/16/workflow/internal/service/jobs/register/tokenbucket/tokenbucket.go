/*
Package tokenbucket registers a job that is used to fetch a token from a token bucket.

Register name: "tokenBucket"
Args:
	"bucket"(mandatory): The name of the bucket
	"fatal"(mandatory): true if a failure should cause a fatal error, false if it should block until it gets one
Result:
	If the site is not in decom, will return a fatal error.
*/
package tokenbucket

import (
	"context"
	"fmt"
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/data/packages/sites"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/service/jobs"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/token"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/proto"
)

var buckets = map[string]*token.Bucket{}

// This registers our Job on server startup.
func init() {
	jobs.Register("tokenBucket", newJob())

	satDiskErase, err := token.New(1, 1, 30*time.Minute)
	if err != nil {
		panic(err)
	}
	buckets["diskEraseSatellite"] = satDiskErase
}

type args struct {
	bucket string
	fatal  bool
}

func (a *args) validate(args map[string]string) error {
	must := map[string]bool{
		"bucket": false,
		"fatal":  false,
	}

	for k, v := range args {
		switch k {
		case "bucket":
			if _, ok := buckets[v]; !ok {
				return fmt.Errorf("bucket(%s) was not a valid", v)
			}
			must["bucket"] = true
			a.bucket = v
		case "fatal":
			switch v {
			case "true":
				a.fatal = true
			case "false":
				a.fatal = true
			default:
				return fmt.Errorf("arg(fatal) was not true or false, was %q", v)
			}
			must["fatal"] = true
		default:
			return fmt.Errorf("invalid arg(%s)", k)
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
	if j.args.fatal {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 1*time.Second)
		defer cancel()
	}
	if err := buckets[j.args.bucket].Token(ctx); err != nil {
		if j.args.fatal {
			return jobs.Fatalf("token(%s) not available", j.args.bucket)
		}
		return jobs.Fatalf("workflow cancelled before token(%s) was available", j.args.bucket)
	}
	return nil
}
