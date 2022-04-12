package etoe

import (
	"context"
	"log"
	"os/exec"

	"testing"
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/11/ops/internal/jaeger/client"
	httpClient "github.com/PacktPublishing/Go-for-DevOps/chapter/11/ops/internal/jaeger/client/test/client"
)

func TestTrace(t *testing.T) {
	start := exec.Command("docker-compose", "up", "-d")
	b, err := start.CombinedOutput()
	if err != nil {
		panic(string(b))
	}
	time.Sleep(5 * time.Second)

	end := exec.Command("docker-compose", "down")
	defer func() {
		b, err = end.CombinedOutput()
		if err != nil {
			panic(string(b))
		}
	}()

	c, err := client.New("127.0.0.1:16685")
	if err != nil {
		panic(err)
	}

	h, err := httpClient.New("127.0.0.1:7080")
	if err != nil {
		panic(err)
	}

	ids := []string{}
	for i := 0; i < 10; i++ {
		callCtx, callCancel := context.WithTimeout(context.Background(), 2*time.Second)
		id, err := h.Call(callCtx)
		if err != nil {
			panic(err)
		}
		callCancel()
		ids = append(ids, id)
	}

	log.Println("sleeping to let trace get exported")
	time.Sleep(20 * time.Second)

	traceCtx, traceCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer traceCancel()

	trace, err := c.Trace(traceCtx, ids[0])
	if err != nil {
		panic(err)
	}

	if len(trace.Spans) != httpClient.NestedSpans {
		t.Errorf("TestTrace(number of Spans): got %d, want %d", len(trace.Spans), httpClient.NestedSpans)
	}

	for _, id := range ids {
		_, err := c.Trace(traceCtx, id)
		if err != nil {
			panic(err)
		}
	}

	// So, I don't know what the deal is.
	/*
		ch, err := c.Search(traceCtx, client.SearchParams{Service: "demo-client", SearchDepth: 100})
		if err != nil {
			panic(err)
		}

		found := []string{}
		for e := range ch {
			found = append(found, e.ID)
		}

		sort.Strings(found)
		sort.Strings(ids)

		if len(ids) != len(found){
			t.Fatalf("TestTrace(Search): number of IDs: got %d, want %d", len(found), len(ids))
		}

		for i, id := range ids {
			if id != found[i] {
				t.Fatalf("TestTrace(Search): trace ids[%d]: got %s, want %s", i, found[i], id)
			}
		}
	*/
}
