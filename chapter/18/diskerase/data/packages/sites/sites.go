/*
Package sites contains types, functions and methods for reading and interpreting data
about sites contained in data files sites.json and machines.json

This data can be accessed through the global variable "Data".
*/
package sites

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var (
	siteNameRE    = regexp.MustCompile(`[a-z][a-z][a-z]`)
	machineNameRE = regexp.MustCompile(`[a-z][a-z]0[0-9]`)
)

// Data holds the site data and is how to access the data.
// Note: in a real system, the data here would always be a copy. That way
// different things accessing it could not change it for others.
var Data SiteData

// Init initializes our Data given the data location. Call from main().
func Init(loc string) {
	sd, err := newSiteData(loc)
	if err != nil {
		panic(err)
	}
	Data = sd
}

// Site represents an individual site where we have machines located.
type Site struct {
	// Name is the name of the site.
	Name string
	// Type is the type of site.
	Type string
	// Status is the status of the site.
	Status string
	// Machines are a list of machines in the site.
	Machines []Machine
}

// Validate validates Site's fields are valid.
func (s Site) Validate() error {
	if !siteNameRE.MatchString(s.Name) {
		return fmt.Errorf(".Name(%s) is not a valid Site name", s.Name)
	}
	switch s.Type {
	case "satellite", "cluster":
	default:
		return fmt.Errorf("site has .Type(%s) that is invalid", s.Type)
	}
	switch s.Status {
	case "inService", "decom", "removed":
	default:
		return fmt.Errorf("site has .Status(%s) that is invalid", s.Status)
	}
	return nil
}

// Machine represents a physical machine located in a site.
type Machine struct {
	// Name is the name of the machine.
	Name string
	// Site is the site the machine is located at.
	Site string
}

// FullName retrieves the globally unique name for the machine.
func (m Machine) FullName() string {
	return m.Name + "." + m.Site
}

// Validate validates a Machine's fields.
func (m Machine) Validate() error {
	if !machineNameRE.MatchString(m.Name) {
		return fmt.Errorf(".Name(%s) is not a valid Machine name", m.Name)
	}
	if !siteNameRE.MatchString(m.Site) {
		return fmt.Errorf(".Site(%s) is not a valid Site name for a Machine to belong to", m.Site)
	}
	return nil
}

// SiteData contains information on our sites and the machines that are located in those sites.
type SiteData struct {
	// Sites is mapping of all sites by name.
	Sites map[string]Site
	// Machines is a mapping of all machines by their full name (aa01.aaa).
	Machines map[string]Machine
}

// newSiteData creates a new SiteData instance by reading sites.json and machines.json from "dir".
func newSiteData(dir string) (SiteData, error) {
	sf, err := os.Open(filepath.Join(dir, "sites.json"))
	if err != nil {
		return SiteData{}, err
	}
	defer sf.Close()

	mf, err := os.Open(filepath.Join(dir, "machines.json"))
	if err != nil {
		return SiteData{}, err
	}
	defer mf.Close()

	sd := SiteData{Sites: map[string]Site{}, Machines: map[string]Machine{}}

	sitesDec := json.NewDecoder(sf)
	for sitesDec.More() {
		s := Site{}
		if err := sitesDec.Decode(&s); err != nil {
			return SiteData{}, err
		}
		if err := s.Validate(); err != nil {
			return SiteData{}, err
		}
		sd.Sites[s.Name] = s
	}

	machinesDec := json.NewDecoder(mf)
	for machinesDec.More() {
		m := Machine{}
		if err := machinesDec.Decode(&m); err != nil {
			return SiteData{}, err
		}
		if err := m.Validate(); err != nil {
			return SiteData{}, err
		}
		s, ok := sd.Sites[m.Site]
		if !ok {
			return SiteData{}, fmt.Errorf("Machine(%s) has Site(%s) that was not found in our sites.json file", m.Name, m.Site)
		}
		s.Machines = append(s.Machines, m)
		sd.Sites[m.Site] = s
		sd.Machines[m.FullName()] = m
	}
	return sd, nil
}
