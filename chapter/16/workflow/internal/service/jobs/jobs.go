/*
Package jobs defines our Job type, which executes work and a registration system for registering
Jobs.

Packages that contain jobs can register themselves by doing:
	func init() {
		jobs.Register("name", job)
	}
If there is a duplicate name, this will panic.

Fetching a Job is simply:
	GetJob(jt string) (Job, error)
*/
package jobs

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/proto"
)

var jobs = map[string]Job{}

// Register registers a Job so that it can be executed.
func Register(name string, job Job) {
	name = strings.TrimSpace(name)
	if name == "" {
		panic("cannot Register empty JobType")
	}
	if _, ok := jobs[name]; ok {
		panic(fmt.Sprintf("cannot register Job(%s) twice", name))
	}
	log.Println("Registered Job: ", name)
	jobs[name] = job
}

// GetJob returns a Job by its type from the registry.
func GetJob(name string) (Job, error) {
	j, ok := jobs[name]
	if !ok {
		return nil, fmt.Errorf("Job(%v) not found", name)
	}
	return j, nil
}

// FatalErr is a an error that should terminate a Workflow.
type FatalErr struct {
	err error
}

// Fatalf creates a fatal error similar to fmt.Errorf().
func Fatalf(msg string, a ...interface{}) FatalErr {
	return FatalErr{err: fmt.Errorf(msg, a...)}
}

// IsFatal indicates if an error is fatal.
func IsFatal(err error) bool {
	return errors.Is(err, FatalErr{})
}

// Is implements the built in Is method.
func (f FatalErr) Is(target error) bool {
	switch target.(type) {
	case FatalErr, *FatalErr:
		return true
	}
	return false
}

// Error() implements error.Error().
func (f FatalErr) Error() string {
	if f.err == nil {
		return ""
	}
	return f.err.Error()
}

// Unwrap implements the built in Unwrap method.
func (f FatalErr) Unwrap() error {
	return errors.Unwrap(f.err)
}

// Job executes some type of work.
type Job interface {
	// Validate validates that the Job settings sent to the server are valid.
	Validate(job *pb.Job) error
	// Run runs the Job settings.
	Run(ctx context.Context, job *pb.Job) error
}
