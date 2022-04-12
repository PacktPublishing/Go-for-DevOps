/*
Package validatedecom registers a job that is used to validate a site is set to state "decom".

Register name: "validateDecom"
Args:
	"site"(mandatory): The name of the site, like "aaa" or "aba"
	"siteType"(mandatory): The type of the site, like "satellite" or "cluster"
Result:
	If the site is not in decom, will return a fatal error.
*/
package validatedecom

import (
	"context"
	"fmt"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/data/packages/sites"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/service/jobs"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/proto"
)

// This registers our Job on server startup.
func init() {
	jobs.Register("validateDecom", newJob(sites.Data.Sites))
}

type args struct {
	site     string
	siteType string
}

func (a *args) validate(args map[string]string) error {
	must := map[string]bool{
		"site": false,
		"type": false,
	}
	var siteData sites.Site

	for k, v := range args {
		switch k {
		case "site":
			s, ok := sites.Data.Sites[v]
			if !ok {
				return fmt.Errorf("site(%s) was not a valid site", v)
			}
			must["site"] = true
			a.site = v
			siteData = s
		case "type":
			must["type"] = true
			a.siteType = v
		default:
			return fmt.Errorf("invalid arg(%s)", k)
		}
	}

	if siteData.Type != a.siteType {
		return fmt.Errorf("site(%s) is type(%s), we expected(%s)", a.site, siteData.Type, a.siteType)
	}

	if siteData.Status != "decom" {
		return fmt.Errorf("site(%s) is not in the decom state, was in %q", a.site, siteData.Status)
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

func newJob(sites map[string]sites.Site) *Job {
	return &Job{sites: sites}
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
	site, ok := sites.Data.Sites[j.args.site]
	if !ok {
		return jobs.Fatalf("site(%s) is no longer in the sites file", j.args.site)
	}

	if site.Status != "decom" {
		return jobs.Fatalf("site(%s) was transitioned out of decom before Job ran", j.args.site)
	}
	return nil
}
