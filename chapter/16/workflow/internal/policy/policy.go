/*
Package policy provides policy primatives, policy registration and functions to run policies against
a WorkReq that is submitted to the system.
*/
package policy

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/proto"
)

var policies = map[string]registration{}

type registration struct {
	policy   Policy
	settings Settings
}

// Setting holds a struct that is used to hold the Policy settings that are invoked.
// These are defined per policy. This should always be a struct and not a *struct.
type Settings interface {
	// Validate validates the Settings for a Policy.
	Validate() error
}

// Register registers a policy by name with an empty Settings that will be copied
// to provide the Settings we will read out of the config file.
func Register(name string, p Policy, s Settings) {
	name = strings.TrimSpace(name)
	if name == "" {
		panic("cannot register a policy with an empty name")
	}

	if _, ok := policies[name]; ok {
		panic(fmt.Sprintf("cannot register two policies with the same name(%s)", name))
	}
	if p == nil {
		panic("cannot register a nil policy")
	}
	if s == nil {
		panic("cannot register a policy with a nil setting")
	}
	if reflect.ValueOf(s).Kind() != reflect.Struct {
		panic(fmt.Sprintf("cannot register a policy(%s) with settings that are not a struct", name))
	}
	log.Println("Registered Policy: ", name)
	policies[name] = registration{policy: p, settings: s}
}

// GetSettings fetches the Settings for a named Policy.
func GetSettings(name string) (Settings, error) {
	r, ok := policies[name]
	if !ok {
		return nil, fmt.Errorf("policy(%s) cannot be found", name)
	}
	return r.settings, nil
}

// Policy represents a policy that is defined to check a WorkReq is compliant.
type Policy interface {
	// Run runs the policy against a request with settings that are specific
	// to the policy. settings can be nil for certain implementations.
	Run(ctx context.Context, req *pb.WorkReq, settings Settings) error
}

// PolicyArgs detail a policy and settings to use to invoke it.
type PolicyArgs struct {
	// Name of the policy in the registry.
	Name string
	// Settings for that policy invocation.
	Settings Settings
}

// Run runs all policies runners that are passed concurrently.
func Run(ctx context.Context, req *pb.WorkReq, args ...PolicyArgs) error {
	if len(args) == 0 {
		return nil
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	// Make a deep clone so that no policy is able to make changes.
	creq := proto.Clone(req).(*pb.WorkReq)

	// Validate that all the policies actually exist and build runners
	// to simply run them against the settings in the next part.
	runners := make([]func() error, 0, len(args))
	for _, arg := range args {
		arg := arg
		r, ok := policies[arg.Name]
		if !ok {
			return fmt.Errorf("policy(%s) does not exist", arg.Name)
		}
		runners = append(
			runners,
			func() error {
				err := r.policy.Run(ctx, creq, arg.Settings)
				if err != nil {
					return fmt.Errorf("policy(%s) violation: %w", arg.Name, err)
				}
				return nil
			},
		)
	}

	wg := sync.WaitGroup{}
	ch := make(chan error, 1)

	// Run all policy invocations concurrently.
	wg.Add(len(runners))
	for _, r := range runners {
		r := r
		go func() {
			defer wg.Done()
			if err := r(); err != nil {
				select {
				case ch <- err:
					cancel()
				default:
				}
				return
			}
		}()
	}
	wg.Wait()

	select {
	case err := <-ch:
		return err
	default:
	}

	if !proto.Equal(req, creq) {
		return fmt.Errorf("a policy tried to modify a request: this is not allowed as it is a security violation")
	}

	return nil
}
