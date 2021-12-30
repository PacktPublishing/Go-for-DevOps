/*
Package es contains an emergency stop implementation. This data is read from es.json file
every 10 seconds. If the data changes, subscribers will receive an update.

Using this is simple:
	ch, cancel := es.Data.Subscribe("SatelliteDiskErase")
	defer cancel()

	if <-ch != es.Go {
		// Do something
	}

	select {
	case <-ch:
		log.Println("ES changed to Stop state ")
	}
*/
package es

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Data is how to access the emergency stop information.
var Data *Reader

func init() {
	d, err := newReader()
	if err != nil {
		panic(err)
	}
	Data = d
}

// Status indicates the emergency stop status.
type Status string

const (
	// Unknown means the status was not set.
	Unknown Status = ""
	// Go indicates the matching workflow can execute.
	Go Status = "go"
	// Stop indicates that the matching workflow should not execute and
	// existing ones should be stopped.
	Stop Status = "stop"
)

// Info is the emergency stop information for a particular entry in our es.json file.
type Info struct {
	// Name is the WorkReq type.
	Name string
	// Status is the emergency stop status.
	Status Status
}

func (i Info) validate() error {
	i.Name = strings.TrimSpace(i.Name)
	if i.Name == "" {
		return fmt.Errorf("es.json: rule with empty name, ignored")
	}
	switch i.Status {
	case "go", "stop":
	default:
		return fmt.Errorf("es.json: rule(%s) has invalid Status(%s), ignored", i.Name, i.Status)
	}
	return nil
}

// Reader reads the es.json file at intervals and makes the data and changes to the data
// available.
type Reader struct {
	entries atomic.Value // map[string]Info

	mu          sync.Mutex
	subscribers map[string][]chan Status
}

func newReader() (*Reader, error) {
	r := &Reader{subscribers: map[string][]chan Status{}}

	m, err := r.load()
	if err != nil {
		return nil, err
	}
	r.entries.Store(m)

	go r.loop()
	return r, nil
}

// Cancel is used to cancel your subscription.
type Cancel func()

// Subscribe returns a channel that sends a Status whenever a ES entry changes status.
// This will send the initial Status immediately. If the name is not found, this will
// send Stop and close the channel. If there is a transition to Stop, the channel will
// be closed once this is sent. Once you either receive a Stop or are no longer interested
// in listening, simply call Cancel().
func (r *Reader) Subscribe(name string) (chan Status, Cancel) {
	i, ok := r.entries.Load().(map[string]Info)[name]
	if !ok || i.Status != Go {
		ch := make(chan Status, 1)
		ch <- Stop
		close(ch)
		return ch, func() {}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	ch := make(chan Status, 1)
	ch <- Go

	l := r.subscribers[name]
	l = append(l, ch)
	r.subscribers[name] = l

	// This removes the channel when it is no longer needed because no
	// one is listening.
	cancel := func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		l := make([]chan Status, 0, len(r.subscribers[name])-1)
		for _, stored := range r.subscribers[name] {
			if stored == ch {
				continue
			}
			l = append(l, stored)
		}
		r.subscribers[name] = l
	}

	return ch, cancel
}

// Status returns the ES status for the named workflow.
func (r *Reader) Status(name string) Status {
	m := r.entries.Load().(map[string]Info)
	switch m[name].Status {
	case Go:
		return Go

	}
	return Stop
}

// loop reads the es.json file in every 10 seconds and updates subscribers of changes
// from Go status to Stop status.
func (r *Reader) loop() {
	for _ = range time.Tick(10 * time.Second) {
		newInfos, err := r.load()
		if err != nil {
			// This means the file was malformed or missing. In these
			// cases we stop all work.
			r.mu.Lock()
			for name := range r.subscribers {
				r.sendStop(name)
			}
			r.mu.Unlock()
			continue
		}
		for name, info := range r.entries.Load().(map[string]Info) {
			newInfo, ok := newInfos[name]
			if !ok {
				r.sendStop(name)
				continue
			}
			if info.Status == Go && newInfo.Status != Go {
				r.sendStop(name)
				continue
			}
		}
		r.entries.Store(newInfos)
	}
}

// sendStop sends a Stop State change to all subscriber to a name.
func (r *Reader) sendStop(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	chans := r.subscribers[name]

	for _, ch := range chans {
		for {
			select {
			case ch <- Stop:
				break
			default:
				// If somehow the channel is full, remove the old entry
				// and then loop and add the most recent one.
				select {
				case <-ch:
				default:
				}
			}
		}
		close(ch)
		delete(r.subscribers, name)
	}
}

// load loads the current es.json values and returns them. Any error is an indication
// that the file could not be read.
func (r *Reader) load() (map[string]Info, error) {
	f, err := os.Open("configs/es.json")
	if err != nil {
		return map[string]Info{}, fmt.Errorf("could not open configs/es.json: %w", err)
	}

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()

	m := map[string]Info{}

	for dec.More() {
		info := Info{}
		if err := dec.Decode(&info); err != nil {
			r.entries.Store(map[string]Info{})
			return map[string]Info{}, fmt.Errorf("es.json file is badly formatted, all jobs moving into stop state")
		}
		if _, ok := m[info.Name]; ok {
			log.Printf("es.json file has two definitions(%s) with the same name, ignoring the second", info.Name)
			continue
		}
		m[info.Name] = info
	}
	return m, nil
}
