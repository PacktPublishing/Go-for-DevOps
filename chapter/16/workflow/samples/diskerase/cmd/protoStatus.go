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
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/client"

	"github.com/fatih/color"
	"github.com/inancgumus/screen"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/16/workflow/proto"
)

// protoStatusCmd represents the protoStatus command
var protoStatusCmd = &cobra.Command{
	Use:   "protoStatus",
	Short: "Streams the status of a workflow in proto's JSON format until it ends",
	Long: `If you have a workflow that you want to monitor the status of,
this will do that. It can be used for more than just diskerase, though
it is primarly meant for that purpose. This differes from "status" in that it outputs
in the full proto JSON format with all data.

Simply pass the single argument, which is the ID of the workflow.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Printf("must pass a single arg, the ID of the workflow to monitor")
			return
		}
		c, err := client.New(rootCmd.Flag("address").Value.String())
		if err != nil {
			fmt.Printf("could not connect to workflow service: %s\n", err)
			return
		}
		if err := monitorProto(context.Background(), c, args[0]); err != nil {
			fmt.Printf("had problem talking to server: %s", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(protoStatusCmd)
}

// monitorProto will contact the server every 10 seconds until the workflow with "id"
// has left the running state.
func monitorProto(ctx context.Context, c *client.Workflow, id string) error {
	for {
		resp, err := c.Status(ctx, id)
		if err != nil {
			return fmt.Errorf("problem getting status of ID(%s): %w", id, err)
		}
		screen.Clear()
		screen.MoveTopLeft()

		color.New(color.FgRed).Println("Updates every 10 seconds")
		fmt.Println(protojson.Format(resp))
		if resp.Status != pb.Status_StatusRunning {
			fmt.Println("Workflow completed!")
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
		}
	}
}
