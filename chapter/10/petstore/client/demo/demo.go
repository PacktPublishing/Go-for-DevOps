package main

import (
	"context"
	_ "embed"
	"log"
	"strings"
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/10/petstore/client"
	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/10/petstore/proto"
	dpb "google.golang.org/genproto/googleapis/type/date"
)

//go:embed names.txt
var namesFile string

func main() {
	names := strings.Split(namesFile, "\n")

	time.Sleep(1 * time.Second)
	c, err := client.New("demo-server:6742")
	if err != nil {
		panic(err)
	}

	start := time.Date(2020, 1, 1, 1, 1, 0, 0, time.UTC)
	t := 1

	ctx := context.Background()

	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			continue
		}
		ids, err := c.AddPets(
			ctx,
			[]*pb.Pet{
				{
					Name: name,
					Type: pb.PetType(t),
					Birthday: &dpb.Date{
						Month: int32(start.Month()),
						Day:   int32(start.Day()),
						Year:  int32(start.Year()),
					},
				},
			},
		)
		if err != nil {
			panic("had an unexpected problem: " + err.Error())
		}
		log.Println("Added pet with ID: ", ids[0])
		t++
		// Only 4 pet types.
		if t == 5 {
			t = 1
		}
		start.Add(24 * time.Hour)
		time.Sleep(500 * time.Millisecond)
	}
}
