/*
Copyright © 2021 John Doak

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/client"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/data/packages/sites"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/proto"

	"github.com/spf13/cobra"
)

const (
	submitLog = "submit.log"
)

// eraseSatelliteCmd represents the eraseSatellite command
var eraseSatelliteCmd = &cobra.Command{
	Use:   "eraseSatellite",
	Short: "Erase all disks at a satellite datacenter",
	Long: `Erase all disks at a satellite datacenter. The satellite must be in
the decom state
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Printf("must pass a single arg, the name of the satellite datacenter")
			return
		}
		sat := args[0]

		fmt.Println("generating workflow...")
		wf, err := generateWork(sat)
		if err != nil {
			fmt.Printf("problem generating workflow: %s\n", err)
			return
		}

		// Open our attempt.log file to write our submissions
		f, err := os.OpenFile(submitLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			fmt.Printf("could not open our submit.log file: %s", err)
			return
		}

		c, err := client.New(rootCmd.Flag("address").Value.String())
		if err != nil {
			fmt.Printf("could not connect to workflow service: %s\n", err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		fmt.Println("submitting workflow...")
		id, err := c.Submit(ctx, wf)
		if err != nil {
			fmt.Printf("submission had an issue: %s\n", err)
			return
		}

		if _, err := f.Write([]byte("\n" + id)); err != nil {
			fmt.Printf("could not write to our submit.log file: %s", err)
			return
		}
		f.Close()

		fmt.Printf("workflow(%s) accepted, ask server to execute workflow...\n", id)

		// Now execute our attempt.
		err = c.Exec(ctx, id)
		if err != nil {
			fmt.Printf("executing workflow(%s) on the server had an issue: %s\n", id, err)
			return
		}
		fmt.Printf("server is executing workflow(%s)\n", id)

		if err := monitor(context.Background(), c, id); err != nil {
			fmt.Printf("problem monitoring workflow(%s): %s", id, err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(eraseSatelliteCmd)
}

// generateWork takes in the satellite name, validates the satellite can have diskerase
// called on it and finally generates the *pb.WorkReq that would be needed to erase that
// satellite's machine's disk.
func generateWork(sat string) (*pb.WorkReq, error) {
	wf := &pb.WorkReq{
		Name: "SatelliteDiskErase",
		Desc: "Erasing disks in datacenter satellite " + sat,
	}

	site, ok := sites.Data.Sites[sat]
	if !ok {
		return nil, fmt.Errorf("there is no datacenter called %q", sat)
	}

	if site.Type != "satellite" {
		return nil, fmt.Errorf("%q is not a satellite datacenter, it is a %s", sat, site.Type)
	}

	if site.Status != "decom" {
		return nil, fmt.Errorf("%q is not in the second state, was in %s", sat, site.Status)
	}

	// Get a list of machines for the site, in alphabetical order.
	machines := make([]sites.Machine, 0, len(site.Machines))
	for _, m := range site.Machines {
		machines = append(machines, m)
	}
	sort.SliceStable(
		machines,
		func(i, j int) bool {
			return site.Machines[i].Name < site.Machines[j].Name
		},
	)

	// This adds a top level block that validates the site is still in the decom state
	// and gets a token from the token bucket to do a satellite erasure.
	preCond := &pb.Block{
		Desc: "Check pre-conditions",
		Jobs: []*pb.Job{
			{
				Name: "validateDecom",
				Desc: fmt.Sprintf("Validate satellite(%s) is in the decom state", sat),
				Args: map[string]string{
					"site": sat,
					"type": "satellite",
				},
			},
			{
				Name: "tokenBucket",
				Desc: "Get disk erase token, which limits our satellite decoms per hour",
				Args: map[string]string{
					"bucket": "diskEraseSatellite",
					"fatal":  "true",
				},
			},
		},
	}
	wf.Blocks = append(wf.Blocks, preCond)

	// For ever set of 5 machines, build a block erasing those 5 machines.
	// Sleep at the end of the block for 1 minute.
	// Set the concurrency to 5 so that all disk erasures in a block happen at the same tiem.
	block := &pb.Block{}
	for i, m := range machines {
		// Every 5 machines, commit the block and start a new one.
		if i%5 == 0 && i != 0 {
			block.Desc = getBlockDesc(block)
			block.RateLimit = 5
			block.Jobs = append(
				block.Jobs,
				&pb.Job{
					Name: "sleep",
					Desc: "Wait 1 minute between disk erasures",
					Args: map[string]string{
						"seconds": "60",
					},
				},
			)
			wf.Blocks = append(wf.Blocks, block)
			block = &pb.Block{}
		}
		block.Jobs = append(
			block.Jobs,
			&pb.Job{
				Name: "diskErase",
				Desc: fmt.Sprintf("Erase satellite(%s) machine(%s) disk", m.Site, m.Name),
				Args: map[string]string{
					"machine": m.Name,
					"site":    m.Site,
				},
			},
		)
	}
	// If we have a block not committed, commit it. Dont' put a sleep afterwards.
	if len(block.Jobs) != 0 {
		block.Desc = getBlockDesc(block)
		block.RateLimit = 5
		wf.Blocks = append(wf.Blocks, block)
	}
	return wf, nil
}

// getBlockDesc writes a description of the block listing the range of
// machines that are currently getting erased.
func getBlockDesc(block *pb.Block) string {
	jobStart := block.Jobs[0].Args["machine"]
	jobEnd := block.Jobs[len(block.Jobs)-1].Args["machine"]
	return fmt.Sprintf("disk erase machines %s-%s", jobStart, jobEnd)
}
