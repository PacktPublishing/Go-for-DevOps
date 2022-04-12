/*
Package config stores the policy configuration Go representation, a global variable called
Policies that is for reading the Config as it is updated on disk and
configuration validation to make sure errors don't slip in to the Config.

A configuration is stored in JSON and looks like:
{
	"Name": "SateliteDiskErase",
	"Policies": [
		{
			"Name": "restrictJobTypes",
			"Settings": {
				"AllowedJobs": [
				        "JTValidateDecom",
        				"JTDiskErase",
        				"JTSleep",
        				"JTGetTokenFromBucket"
				]
			}
		}
	]
}
...
*/
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/internal/policy"
)

// Policies provides the policy Reader that can be used to read the current policy.
var Policies *Reader

// Init is called in main to initialize our reads of the policy file. It is called
// manually instead of init() to guarantee other init() statements are run first.
func Init() {
	r, err := newReader("configs/policies.json")
	if err != nil {
		panic(err)
	}
	Policies = r
}

// Config represents a policy config that is stored on disk.
type Config struct {
	// Workflows stored by name.
	Workflows map[string]Workflow

	// Set if the last load of a new Config had errors. This Config will not
	// have any errors if this is set.
	err error
}

func newConfig() Config {
	return Config{
		Workflows: map[string]Workflow{},
	}
}

func (c Config) validate() error {
	for k, w := range c.Workflows {
		if err := w.validate(); err != nil {
			return err
		}
		c.Workflows[k] = w // In case there were any changes.
	}
	return nil
}

// Workflow stores a Workflow name and the Policies for that Workflow.
type Workflow struct {
	// Name is the name of the Workflow.
	Name string
	// Policies are the Policies to be applied to that Workflow.
	Policies []Policy
}

func (w Workflow) validate() error {
	w.Name = strings.TrimSpace(w.Name)
	if w.Name == "" {
		return fmt.Errorf("Workflow cannot have an empty Name field")
	}

	for i, p := range w.Policies {
		if err := p.validate(); err != nil {
			return fmt.Errorf("Workflow(%s): %s", w.Name, err)
		}
		w.Policies[i] = p // Stores the Policy that has SettingsTyped stored
	}
	return nil
}

// Policy is the policy to apply to a Workflow.
type Policy struct {
	// Name is the name of the policy type.
	Name string
	// Settings are the settings for that particular policy. We store this as
	// a RawMessage so that we can parse it into the typed version on a second pass.
	Settings json.RawMessage
	// SettingsTyped is the parsed version of the Settings. This is not stored in the Config.
	SettingsTyped policy.Settings `json:"-"`
}

func (p *Policy) validate() error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return fmt.Errorf("Policy cannot have an empty Name field")
	}
	s, err := policy.GetSettings(p.Name)
	if err != nil {
		return err
	}

	// This section is going to be confusing, as it is using an advanced topic called
	// runtime reflection. This is a topic that really can be its own book. Suffice it to say,
	// I wanted every registered implementation of our policy.Settings to be a struct{}, not
	// a *struct. This makes it easy to make copies of it without worrying about accidental
	// modification. But, you can only unmarshal JSON into a *struct. Because all these specific
	// Settings are stored inside an interface called policy.Settings, you can't do:
	// &s, because that would be the address of the interface, not the underlying value.
	// Confused?? Yeah, I know.......
	// So here we are going to use the reflect package to create a pointer to the specific
	// value the user put in the interface. Then we are going to unmarshal into that.
	// Then we are going to convert that back to a policy.Settings interface.
	// Don't spend a bunch of time here, reflection is something better left avoided if you can.
	val := reflect.ValueOf(s)
	ptr := reflect.New(val.Type())

	if err := json.Unmarshal(p.Settings, ptr.Interface()); err != nil {
		return fmt.Errorf("policy(%s) could not unmarshal its Settings: %s", p.Name, err)
	}
	p.SettingsTyped = ptr.Elem().Interface().(policy.Settings)

	if err := p.SettingsTyped.Validate(); err != nil {
		return fmt.Errorf("policy(%s) Settings did not validate: %s", p.Name, err)
	}

	return nil
}

// Reader is used to read the current policy configuration. The
// configuration is checked for updates every 10 seconds and if there
// is a valid configuration, it is updated. If not, an error is recorded.
// Once a Reader is returned by New(), it guarantees to always return a Config.
// If there is an error also returned, then the Config is the last known good
// Config.
type Reader struct {
	loc  string
	conf atomic.Value // Config
}

// Read reads the latest Config we have. If an error is returned, the Config will
// be valid, but it will be the last known good Config instead of the broken latest
// Config.
func (r *Reader) Read() (Config, error) {
	c := r.conf.Load().(Config)
	return c, c.err
}

func (r *Reader) update() {
	for _ = range time.Tick(10 * time.Second) {
		if err := r.load(); err != nil {
			c := r.conf.Load().(Config)
			c.err = err
			r.conf.Store(c)
		}
	}
}

func (r *Reader) load() error {
	f, err := os.Open(r.loc)
	if err != nil {
		return fmt.Errorf("cannot access policy config(%s): %w", r.loc, err)
	}
	defer f.Close()

	c := newConfig()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	for dec.More() {
		w := Workflow{}
		if err := dec.Decode(&w); err != nil {
			return fmt.Errorf("policy on disk could not be JSON decoded: %w", err)
		}
		w.Name = strings.TrimSpace(w.Name)
		if w.Name == "" {
			return fmt.Errorf("Workflow cannot have an empty Name field")
		}
		if _, ok := c.Workflows[w.Name]; ok {
			return fmt.Errorf("cannot have two sets of Workflow policies for %q", w.Name)
		}
		c.Workflows[w.Name] = w
	}

	if err := c.validate(); err != nil {
		return fmt.Errorf("policy config had an error: %s", err)
	}
	r.conf.Store(c)
	return nil
}

// newReader returns a Reader that can grab the latest Config on disk.
func newReader(loc string) (*Reader, error) {
	r := &Reader{loc: loc}
	if err := r.load(); err != nil {
		return nil, err
	}
	go r.update()

	return r, nil
}
