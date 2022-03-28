package main

import (
	"encoding/json"
	"os"
)

type site struct {
	Name   string
	Type   string
	Status string
}

type machine struct {
	Name string
	Site string
}

func main() {
	first := byte(97)
	second := byte(97)
	third := byte(97)

	sites := []site{}
	for i := 0; i < 100; i++ {
		t := "cluster"
		if i%3 == 0 {
			t = "satellite"
		}
		site := site{
			Name:   string(append([]byte{}, first, second, third)),
			Type:   t,
			Status: "inService",
		}
		switch site.Name {
		case "aap", "adg", "adv":
			site.Status = "decom"
		}
		sites = append(sites, site)

		if third < 122 {
			third++
		} else {
			third = 97
			second++
			if second == 122 {
				second = 97
				first++
			}
		}
	}
	machines := []machine{}
	for _, site := range sites {
		if site.Type == "satellite" {
			first := byte(97)
			second := byte(97)
			a := byte(48)
			b := byte(48)
			for i := 0; i < 50; i++ {
				name := append([]byte{}, first, second, a, b)
				machines = append(
					machines,
					machine{Name: string(name), Site: site.Name},
				)
				b++
				if b > 57 {
					a = 48
					b = 48
					second++
					if second > 122 {
						first++
						second = 97
					}
				}
			}
		}
	}

	sitef, err := os.OpenFile("sites.json", os.O_WRONLY+os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	defer sitef.Close()
	machinef, err := os.OpenFile("machines.json", os.O_WRONLY+os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	defer machinef.Close()

	encSites := json.NewEncoder(sitef)
	for _, s := range sites {
		if err := encSites.Encode(s); err != nil {
			panic(err)
		}
	}
	encMachines := json.NewEncoder(machinef)
	for _, m := range machines {
		if err := encMachines.Encode(m); err != nil {
			panic(err)
		}
	}
}
