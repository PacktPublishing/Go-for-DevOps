package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/11/petstore/client"
	"google.golang.org/protobuf/encoding/protojson"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/11/petstore/proto"
	dpb "google.golang.org/genproto/googleapis/type/date"
)

var addr = flag.String("addr", "127.0.0.1:6742", "The host:port to connect to")

const helpText = `
Petstore CLI Client Help

Command Add:
	Adds pets to the petstore and returns a list of IDs.

	Syntax:
		petstore add [pet description in JSON]
	Example:
		petstore add '{"name":"Stevie Nicks", "type":"PTFeline", "birthday": {"month": 6, "day": 1, "year": 2005}}'

Command Delete:
	Deletes pets from the petstore.

	Syntax:
		petstore delete [id] [id] [id] ...
	Example:
		petstore delete 62809742-2de1-4208-a8cc-df485c48c563 83968fb4-9502-4df1-8680-7691fc1d3abe

Command Search:
	Searches for pets in the petstore.

	Syntax:
		petstore search param="value" parame="value"
	Params:
		Names - Comma separated list of names
		Type - Comma separated list of pet types	
		BirthdayStart - JSON version of proto date
		BirthdayEnd - JSON version of proto date

		Note: 
		If BirthdayStart is provided by not BirthdayEnd, it will
		be set to the current date + 1 day. If the reverse,
		BirthdayStart will be set to the Go's zero time.
		      
	Example:
		petstore search names="Stevie Nicks, Frank" types="PTFeline" birthdayStart='{"month":1, "day":1, "year":2004}'
`

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if len(os.Args) < 2 {
		fmt.Println("Error: arguments are not valid")
		fmt.Println(helpText)
		os.Exit(1)
	}

	c, err := client.New(*addr)
	if err != nil {
		fmt.Printf("Error: problem connecting to server: %s\n", err)
		os.Exit(1)
	}

	cmd := strings.ToLower(os.Args[1])
	switch cmd {
	case "add":
		if len(os.Args) < 3 {
			fmt.Println("Error: not enough arguments to add command")
			fmt.Println(helpText)
			os.Exit(1)
		}
		p := &pb.Pet{}
		j := os.Args[2]
		if err := protojson.Unmarshal([]byte(j), p); err != nil {
			fmt.Printf("Error: problem with your pet description: %s\n", err)
			os.Exit(1)
		}
		ids, err := c.AddPets(ctx, []*pb.Pet{p})
		if err != nil {
			fmt.Printf("Error: problem adding your pets: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s was added as %s\n", p.Name, ids[0])
		return
	case "delete":
		if len(os.Args) < 3 {
			fmt.Println("Error: not enough arguments to delete command")
			fmt.Println(helpText)
			os.Exit(1)
		}
		if err := c.DeletePets(ctx, os.Args[2:]); err != nil {
			fmt.Printf("Error: problem deleting: %s\n", err)
			os.Exit(1)
		}
		fmt.Println("Delete succeeded")
		return
	case "search":
		if len(os.Args) < 3 {
			fmt.Println("Error: not enough arguments to search command")
			fmt.Println(helpText)
			os.Exit(1)
		}
		r := getSearchReq()
		ch, err := c.SearchPets(ctx, r)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		for p := range ch {
			if p.Error() != nil {
				fmt.Println(p.Error())
				os.Exit(1)
			}
			fmt.Println(protojson.Format(p))
		}
	case "help":
		fmt.Println(helpText)
	default:
		fmt.Println("Error: unknown command: ", cmd)
		fmt.Println(helpText)
	}
}

func getSearchReq() *pb.SearchPetsReq {
	argsSeen := map[string]bool{
		"names":         false,
		"types":         false,
		"birthdayStart": false,
		"birthdayEnd":   false,
	}

	r := &pb.SearchPetsReq{}

	for _, arg := range os.Args[2:] {
		switch {
		case strings.HasPrefix(arg, "names"):
			if argsSeen["names"] {
				fmt.Println("cannot have multiple 'names' parameters")
				os.Exit(1)
			}
			sp := strings.Split(arg, "=")
			if len(sp) != 2 {
				fmt.Println("names parameter is malformed")
				os.Exit(1)
			}
			names := strings.Trim(sp[1], `"`)
			sp = strings.Split(names, ",")
			for _, name := range sp {
				r.Names = append(r.Names, strings.TrimSpace(name))
			}
			argsSeen["names"] = true
		case strings.HasPrefix(arg, "types"):
			if argsSeen["types"] {
				fmt.Println("cannot have multiple 'types' parameters")
				os.Exit(1)
			}
			sp := strings.Split(arg, "=")
			if len(sp) != 2 {
				fmt.Println("types parameter is malformed")
				os.Exit(1)
			}
			types := strings.Trim(sp[1], `"`)
			sp = strings.Split(types, ",")
			for _, t := range sp {
				t = strings.TrimSpace(t)
				e, ok := pb.PetType_value[t]
				if !ok {
					fmt.Printf("types parameter had value %q that we do not recognize\n", t)
				}
				r.Types = append(r.Types, pb.PetType(e))
			}
			argsSeen["types"] = true
		case strings.HasPrefix(arg, "birthdayStart"):
			if argsSeen["birthdayStart"] {
				fmt.Println("cannot have multiple 'birthdayStart' parameters")
				os.Exit(1)
			}
			sp := strings.Split(arg, "=")
			if len(sp) != 2 {
				fmt.Println("birthdayStart parameter is malformed")
				os.Exit(1)
			}
			start := strings.Trim(sp[1], `"`)
			d := &dpb.Date{}
			if err := protojson.Unmarshal([]byte(start), d); err != nil {
				fmt.Printf("birthdayStart parameter is malformed: %s\n", err)
				os.Exit(1)
			}
			if r.BirthdateRange == nil {
				r.BirthdateRange = &pb.DateRange{
					Start: d,
				}
			} else {
				r.BirthdateRange.Start = d
			}
			argsSeen["birthdayStart"] = true
		case strings.HasPrefix(arg, "birthdayEnd"):
			if argsSeen["birthdayEnd"] {
				fmt.Println("cannot have multiple 'birthdayEnd' parameters")
				os.Exit(1)
			}
			sp := strings.Split(arg, "=")
			if len(sp) != 2 {
				fmt.Println("birthdayEnd parameter is malformed")
				os.Exit(1)
			}
			end := strings.Trim(sp[1], `"`)
			d := &dpb.Date{}
			if err := protojson.Unmarshal([]byte(end), d); err != nil {
				fmt.Printf("birthdayEnd parameter is malformed: %s\n", err)
				os.Exit(1)
			}
			if r.BirthdateRange == nil {
				r.BirthdateRange = &pb.DateRange{
					End: d,
				}
			} else {
				r.BirthdateRange.End = d
			}
			argsSeen["birthdayEnd"] = true
		default:
			fmt.Printf("parameter %q is one we don't support\n", arg)
			os.Exit(1)
		}

		// If one of the birthday ranges was put in (start or end) but not the other, add the other.
		if r.BirthdateRange != nil {
			rng := r.BirthdateRange
			if rng.Start != nil && rng.End == nil {
				t := time.Now().Add(24 * time.Hour)
				rng.End = &dpb.Date{Month: int32(t.Month()), Day: int32(t.Day()), Year: int32(t.Year())}
			}
			if rng.End != nil && rng.Start == nil {
				t := time.Time{}
				rng.Start = &dpb.Date{Month: int32(t.Month()), Day: int32(t.Day()), Year: int32(t.Year())}
			}
		}
	}
	return r
}
