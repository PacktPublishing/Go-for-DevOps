package prom

import (
	"context"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	//"github.com/prometheus/common/config"
)

// Client is a wrapper around the prometheus Go client for our needs.
type Client struct {
	client v1.API
}

// New creates a new client connecting to the HTTP address provided. This should be in the form of http[s]://[domain,host,ip]:[port] .
func New(httpAddr string) (*Client, error) {
	client, err := api.NewClient(
		api.Config{
			Address: httpAddr,
		},
	)
	if err != nil {
		return nil, err
	}

	return &Client{client: v1.NewAPI(client)}, nil
}

// Metric returns the metrics current value.
func (c *Client) Metric(ctx context.Context, metric string) (model.Value, v1.Warnings, error) {
	return c.client.Query(ctx, metric, time.Now())
}

// Range does a query within a range of time and returns the result. A query might be something like:
// "rate(prometheus_tsdb_head_samples_appended_total[5m])".
func (c *Client) Range(ctx context.Context, query string, r v1.Range) (model.Value, v1.Warnings, error) {
	return c.client.QueryRange(ctx, query, r)
}

// AlertFilter represents a filter you can use to filter out alerts.
type AlertFilter struct {
	// Labels filters alerts by attached labels. If only a key is provided, we only match on the key.
	// Values that start with "regexp/" are regexp compiled and then matched against a value stored at that key.
	// Defaults to all labels.
	Labels       map[string]string
	labelRegexes map[string]*regexp.Regexp
	// ActiveAt filters out any alerts that are before this time. Defaults to all alerts.
	ActiveAt time.Time
	// States filters alerts to only ones in these states. Defaults to all states.
	States []string
	states map[string]bool
	// Value filters out all values that don't match the regex.
	Value *regexp.Regexp

	compiled bool
}

// Compile compiles the AlertFilter. If this is a one off query, no need to do this. If you plan
// to reuse the filter, this will increase the speed. Changing a filter after Compile() is called
// will not give you the desired result, create a new filter instead.
func (a *AlertFilter) Compile() error {
	if a.compiled {
		return nil
	}
	if len(a.Labels) > 0 {
		regexes := make(map[string]*regexp.Regexp, len(a.Labels))
		for k, v := range a.Labels {
			if strings.TrimSpace(k) == "" {
				return fmt.Errorf("cannot have a empty string label key")
			}
			if strings.HasPrefix(v, "regexp/") {
				sp := strings.Split(v, "regexp/")
				if len(sp) == 1 {
					return fmt.Errorf("label value with regexp/ must have more content")
				}
				if len(sp) > 2 {
					return fmt.Errorf("regexp/ can only be at the beginning of a label value")
				}
				r, err := regexp.Compile(sp[1])
				if err != nil {
					return fmt.Errorf("label with value(%s) cannot be regexp compiled: %w", v, err)
				}
				regexes[k] = r
			}
		}
		a.labelRegexes = regexes
	}
	if len(a.States) > 0 {
		a.states = map[string]bool{}
		for _, s := range a.States {
			a.states[s] = true
		}
	}
	a.compiled = true
	return nil
}

func (a *AlertFilter) filter(items []v1.Alert) (chan v1.Alert, error) {
	if !a.compiled {
		if err := a.Compile(); err != nil {
			return nil, err
		}
	}

	ch := make(chan v1.Alert, runtime.NumCPU())
	limit := make(chan struct{}, runtime.NumCPU())
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, item := range items {
			item := item
			limit <- struct{}{}
			wg.Add(1)
			func() {
				defer wg.Done()
				defer func() { <-limit }()
				go a.pipeline(item, ch)
			}()
		}
	}()

	go func() {
		wg.Wait()
		close(ch)
	}()
	return ch, nil

}

func (a *AlertFilter) pipeline(item v1.Alert, out chan v1.Alert) {
	if len(a.Labels) > 0 {
		if !a.matchLabel(item) {
			return
		}
	}
	if !a.ActiveAt.IsZero() {
		if item.ActiveAt.Before(a.ActiveAt) {
			return
		}
	}
	if len(a.States) > 0 {
		if !a.states[string(item.State)] {
			return
		}
	}

	if a.Value != nil {
		if !a.Value.MatchString(item.Value) {
			return
		}
	}
	out <- item
}

func (a *AlertFilter) matchLabel(item v1.Alert) bool {
	for k, v := range item.Labels {
		matched, ok := a.Labels[string(k)]
		if !ok {
			continue
		}

		// Exact match or we aren't matching on values.
		switch string(v) {
		case "", matched:
			return true
		}

		// Let's see if this was a regex match.
		r := a.labelRegexes[string(k)]
		// It wasn't, so return false because we didn't have an exact match.
		if r == nil {
			continue
		}
		if r.MatchString(string(v)) {
			return true
		}
	}
	return false
}

// Alerts will return all the alerts that match the filter.
func (c *Client) Alerts(ctx context.Context, filter AlertFilter) (chan v1.Alert, error) {
	r, err := c.client.Alerts(ctx)
	if err != nil {
		return nil, err
	}
	return filter.filter(r.Alerts)
}
