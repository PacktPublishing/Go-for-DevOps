package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/15/pkg/helpers"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/15/pkg/mgmt"
)

func init() {
	_ = godotenv.Load()
}

func main() {
	subscriptionID := helpers.MustGetenv("AZURE_SUBSCRIPTION_ID")
	sshPubKeyPath := helpers.MustGetenv("SSH_PUBLIC_KEY_PATH")
	factory := mgmt.NewVirtualMachineFactory(subscriptionID, sshPubKeyPath)
	fmt.Println("Staring to build Azure resources...")
	stack := factory.CreateVirtualMachineStack(context.Background(), "southcentralus")

	var (
		admin           = stack.VirtualMachine.Properties.OSProfile.AdminUsername
		ipAddress       = stack.PublicIP.Properties.IPAddress
		sshIdentityPath = strings.TrimRight(sshPubKeyPath, ".pub")
	)
	fmt.Printf("Connect with: `ssh -i %s %s@%s`\n\n", sshIdentityPath, *admin, *ipAddress)
	fmt.Println("Press enter to delete the infrastructure.")
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')
	factory.DestroyVirtualMachineStack(context.Background(), stack)
}
