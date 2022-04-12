package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/rollout/lb/client"

	"github.com/fatih/color"
	"github.com/rodaine/table"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/8/rollout/lb/proto"
)

var (
	keyFile = flag.String("keyFile", "", "The key file to use for SSH connections. If not set, uses the SSH agent.")
)

var (
	headerFmt = color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt = color.New(color.FgYellow).SprintfFunc()
)

func main() {
	flag.Parse()

	ctx := context.Background()

	config, wf, err := setup()
	if err != nil {
		color.Red("Setup Error: %s", err)
		os.Exit(1)
	}

	err = getSSHConfig(config)
	if err != nil {
		color.Red("SSH setup error: %s", err)
		os.Exit(1)
	}

	// If the load balancer doesn't have pool "/", set one up.
	if _, err := wf.lb.PoolHealth(ctx, "/", false, false); err != nil {
		err := wf.lb.AddPool(
			ctx,
			"/",
			pb.PoolType_PT_P2C,
			client.HealthChecks{
				HealthChecks: []client.HealthCheck{
					client.StatusCheck{
						URLPath:       "/healthz",
						HealthyValues: []string{"ok", "OK"},
					},
				},
				Interval: 5 * time.Second,
			},
		)
		if err != nil {
			color.Red("LB did not have pool `/` and couldn't create it: %s", err)
			os.Exit(1)
		}
		color.Blue("Setup LB with pool `/`")
	}

	color.Red("Starting Workflow")
	if err := wf.run(ctx); err != nil {
		status := wf.status()
		color.Red("Workflow Failed: %s", status.endState)

		var tbl table.Table
		if status.endState == esPreconditionFailure {
			tbl = table.New("Failed State", "Error")
			tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)
			tbl.AddRow("Precondition", err)
		} else {
			tbl = table.New("Endpoint", "Failed State", "Error")
			tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)
			for _, action := range status.failures {
				tbl.AddRow(action.endpoint, action.failure(), action.err)
			}
		}
		tbl.Print()
		os.Exit(1)
	}

	status := wf.status()
	if len(status.failures) == 0 {
		color.Blue("Workflow Completed with no failures")
		os.Exit(0)
	}

	color.Blue("Workflow Completed, but had %d failed actions", len(status.failures))
	for i := 0; i < 3; i++ {
		color.Green("Retrying failed actions in 5 minutes...")
		time.Sleep(5 * time.Minute)
		fmt.Println("Executing failed actions...")

		wf.retryFailed(ctx)
		status = wf.status()
		if len(status.failures) == 0 {
			break
		}
		color.Blue("Workflow Failures retry, but had %d failed actions", len(status.failures))
	}
	status = wf.status()
	if len(status.failures) == 0 {
		color.Blue("Workflow Completed with no failures")
		os.Exit(0)
	}

	color.Blue("Workflow Completed but with %d failures after retries exhausted")
	tbl := table.New("Endpoint", "Failed State", "Error")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for _, action := range status.failures {
		failure := action.failure()
		if failure == "" {
			failure = "during setup"
		}
		tbl.AddRow(action.endpoint, failure, action.err)
	}
	tbl.Print()
	os.Exit(0)
}

func setup() (*config, *workflow, error) {
	if len(flag.Args()) != 1 {
		return nil, nil, fmt.Errorf("must have argument to service file")
	}

	b, err := os.ReadFile(flag.Args()[0])
	if err != nil {
		return nil, nil, fmt.Errorf("can't open workflow configuration file: %w", err)
	}

	config := &config{}
	if err := json.Unmarshal(b, config); err != nil {
		return nil, nil, fmt.Errorf("%q is misconfigured: %w", flag.Args()[0], err)
	}
	if err := config.validate(); err != nil {
		log.Println(string(b))
		return nil, nil, fmt.Errorf("config file didn't validate: %w", err)
	}

	lb, err := client.New(config.LB)
	if err != nil {
		return nil, nil, fmt.Errorf("can't connected to LB(%s): %s\n", config.LB, err)
	}
	wf, err := newWorkflow(config, lb)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create workflow: %w", err)
	}
	return config, wf, nil
}

func getSSHConfig(config *config) error {
	auth, err := getAuthFromFlags()
	if err != nil {
		return err
	}
	if config.BackendUser == "" {
		config.BackendUser = os.Getenv("USER")
	}
	config.ssh = &ssh.ClientConfig{
		User:            config.BackendUser,
		Auth:            []ssh.AuthMethod{auth},
		Timeout:         5 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return nil
}

func getAuthFromFlags() (ssh.AuthMethod, error) {
	if *keyFile != "" {
		return publicKey(*keyFile)
	}
	return agentAuth()
}

func agentAuth() (ssh.AuthMethod, error) {
	conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return nil, fmt.Errorf("problem dialing SSH agent when --key was not provided: %w", err)
	}

	client := agent.NewClient(conn)
	return ssh.PublicKeysCallback(client.Signers), nil
}

func publicKey(privateKeyFile string) (ssh.AuthMethod, error) {
	k, err := os.ReadFile(privateKeyFile)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(k)
	if err != nil {
		return nil, err
	}

	return ssh.PublicKeys(signer), nil
}
