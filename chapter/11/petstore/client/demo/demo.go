/*
The demo will take a list of pet names and insert them into the Petstore every 1/2
a second. At the same time, starting 10 seconds after starting to add pets, another
goroutine will start and begin random searching for pets in the name file.
It will do this len(names) times. Because this is random, sometimes this will be
an error (because the pet hasn't been added yet) and sometimes a success.
The longer this goes on, the less likely there will be an error.
*/
package main

import (
	"context"
	_ "embed"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/11/petstore/client"
	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/11/petstore/proto"
	dpb "google.golang.org/genproto/googleapis/type/date"
)

//go:embed names.txt
var namesFile string

func main() {
	ctx := context.Background()

	time.Sleep(1 * time.Second)
	c, err := client.New("petstore:6742")
	if err != nil {
		panic(err)
	}

	names := strings.Split(namesFile, "\n")

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		addNames(ctx, c, names)
	}()

	time.Sleep(10 * time.Second)

	wg.Add(1)
	go func() {
		defer wg.Done()
		searchNames(ctx, c, names)
	}()

	wg.Wait()
}

func addNames(ctx context.Context, c *client.Client, names []string) {
	start := time.Date(2020, 1, 1, 1, 1, 0, 0, time.UTC)
	t := 1

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

func searchNames(ctx context.Context, c *client.Client, names []string) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	l := len(names)

	for i := 0; i < len(names); i++ {
		x := r.Intn(l)
		ch, err := c.SearchPets(ctx, &pb.SearchPetsReq{Names: []string{names[x]}})
		if err != nil {
			log.Fatalf("Search(%s): bad error: %s", names[x], err)
		}
		var results []client.Pet
		for result := range ch {
			results = append(results, result)
		}
		if len(results) > 0 {
			log.Printf("Search(%s): found", names[x])
		} else {
			log.Printf("Search(%s): pet not found", names[x])
		}
		time.Sleep(1 * time.Second)
	}
}
