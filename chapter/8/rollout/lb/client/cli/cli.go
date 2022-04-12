package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/rollout/lb/client"

	"github.com/fatih/color"
	"github.com/rodaine/table"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/8/rollout/lb/proto"
)

var (
	server  = flag.String("lb", "", "The load balancer address to connect to, host:port")
	ip      = flag.String("ip", "", "An IP setting")
	port    = flag.Int("port", 0, "A port setting")
	urlPath = flag.String("url_path", "", "The url path to use")
	pattern = flag.String("pattern", "", "A pattern setting")
)

var hcs = client.HealthChecks{
	Interval: 10 * time.Second,
	HealthChecks: []client.HealthCheck{
		client.StatusCheck{
			URLPath:       "/healthz",
			HealthyValues: []string{"ok", "OK"},
		},
	},
}

func main() {
	flag.Parse()

	if len(flag.Args()) != 1 {
		panic("bad args")
	}

	c, err := client.New(*server)
	if err != nil {
		panic(err)
	}

	switch flag.Args()[0] {
	case "addPool":
		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
		if err := c.AddPool(ctx, *pattern, pb.PoolType_PT_P2C, hcs); err != nil {
			panic(err)
		}
	case "removePool":
		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
		if err := c.RemovePool(ctx, *pattern); err != nil {
			panic(err)
		}
	case "addBackend":
		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)

		b := client.IPBackend{
			IP:      net.ParseIP(*ip),
			Port:    int32(*port),
			URLPath: *urlPath,
		}
		if err := c.AddBackend(ctx, *pattern, b); err != nil {
			panic(err)
		}
	case "removeBackend":
		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)

		b := client.IPBackend{
			IP:      net.ParseIP(*ip),
			Port:    int32(*port),
			URLPath: *urlPath,
		}
		if err := c.AddBackend(ctx, *pattern, b); err != nil {
			panic(err)
		}
	case "poolHealth":
		ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
		ph, err := c.PoolHealth(ctx, *pattern, true, true)
		if err != nil {
			panic(err)
		}
		headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
		columnFmt := color.New(color.FgYellow).SprintfFunc()

		tbl := table.New("Pool", "Status")
		tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)
		tbl.AddRow(*pattern, ph.Status)
		tbl.Print()

		tbl = table.New("Backend", "Status")
		tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)
		for _, b := range ph.Backends {
			switch {
			case b.Backend.GetIpBackend() != nil:
				v := b.Backend.GetIpBackend()
				tbl.AddRow(
					fmt.Sprintf("%s:%d%s", v.Ip, v.Port, v.UrlPath),
					b.Status.String(),
				)
			}
		}
		tbl.Print()
	default:
		panic("non-recognized command")
	}
}
