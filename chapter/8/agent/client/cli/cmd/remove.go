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
	"log"
	"os"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/agent/client"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/8/agent/proto"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove [remote endpoint] [package name]",
	Short: "Remove an application installed by the system agent.",
	Long: `Remove an application installed by the system agent."
An usage example:

cli remove 22.47.60.3:22 helloworld
`,
	Run: func(cmd *cobra.Command, args []string) {
		auth, err := getAuthFromFlags()
		if err != nil {
			log.Println("Error: failed to get SSH authorizaion: ", err)
			os.Exit(1)
		}

		c, err := client.New(
			args[0],
			[]ssh.AuthMethod{auth},
		)

		_, err = c.Remove(
			context.Background(),
			&pb.RemoveReq{
				Name: args[1],
			},
		)
		if err != nil {
			log.Println("Error: ", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// removeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// removeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
