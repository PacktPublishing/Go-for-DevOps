package mgmt

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/mitchellh/go-homedir"
	"github.com/yelinaung/go-haikunator"

	. "github.com/PacktPublishing/Go-for-DevOps/chapter/15/pkg/helpers"
)

var (
	haiku = haikunator.New(time.Now().UnixNano())
)

type VirtualMachineFactory struct {
	subscriptionID string
	sshPubKeyPath  string
	cred           azcore.TokenCredential
	groupsClient   *armresources.ResourceGroupsClient
	vmClient       *armcompute.VirtualMachinesClient
	vnetClient     *armnetwork.VirtualNetworksClient
	subnetClient   *armnetwork.SubnetsClient
	nicClient      *armnetwork.InterfacesClient
	nsgClient      *armnetwork.SecurityGroupsClient
	pipClient      *armnetwork.PublicIPAddressesClient
}

// NewVirtualMachineFactory instantiates an Azure VirtualMachine factory
func NewVirtualMachineFactory(subscriptionID, sshPubKeyPath string) *VirtualMachineFactory {
	cred := HandleErrWithResult(azidentity.NewDefaultAzureCredential(nil))
	return &VirtualMachineFactory{
		cred:           cred,
		subscriptionID: subscriptionID,
		sshPubKeyPath:  sshPubKeyPath,
		groupsClient:   BuildClient(subscriptionID, cred, armresources.NewResourceGroupsClient),
		vmClient:       BuildClient(subscriptionID, cred, armcompute.NewVirtualMachinesClient),
		vnetClient:     BuildClient(subscriptionID, cred, armnetwork.NewVirtualNetworksClient),
		subnetClient:   BuildClient(subscriptionID, cred, armnetwork.NewSubnetsClient),
		nsgClient:      BuildClient(subscriptionID, cred, armnetwork.NewSecurityGroupsClient),
		nicClient:      BuildClient(subscriptionID, cred, armnetwork.NewInterfacesClient),
		pipClient:      BuildClient(subscriptionID, cred, armnetwork.NewPublicIPAddressesClient),
	}
}

type VirtualMachineStack struct {
	Location         string
	sshKeyPath       string
	name             string
	ResourceGroup    armresources.ResourceGroup
	VirtualNetwork   armnetwork.VirtualNetwork
	SecurityGroup    armnetwork.SecurityGroup
	VirtualMachine   armcompute.VirtualMachine
	NetworkInterface armnetwork.Interface
	PublicIP         armnetwork.PublicIPAddress
}

// CreateVirtualMachineStack creates a virtual machine and networking within a resource group
func (vmf *VirtualMachineFactory) CreateVirtualMachineStack(ctx context.Context, location string) *VirtualMachineStack {
	stack := &VirtualMachineStack{
		Location:   location,
		name:       haiku.Haikunate(),
		sshKeyPath: HandleErrWithResult(homedir.Expand(vmf.sshPubKeyPath)),
	}

	stack.ResourceGroup = vmf.createResourceGroup(ctx, stack.name, stack.Location)
	stack.SecurityGroup = vmf.createSecurityGroup(ctx, stack.name, stack.Location)
	stack.VirtualNetwork = vmf.createVirtualNetwork(ctx, stack)
	stack.VirtualMachine = vmf.createVirtualMachine(ctx, stack)
	stack.NetworkInterface = vmf.getFirstNetworkInterface(ctx, stack)
	stack.PublicIP = vmf.getPublicIPAddress(ctx, stack)
	return stack
}

// DestroyVirtualMachineStack deletes a virtual machine and networking within a resource group.
// This function does not wait for completion. Once the delete operation is accepted, the function returns.
func (vmf *VirtualMachineFactory) DestroyVirtualMachineStack(ctx context.Context, vmStack *VirtualMachineStack) {
	_, err := vmf.groupsClient.BeginDelete(ctx, vmStack.name, nil)
	HandleErr(err)
}

// createResourceGroup creates an Azure resource by name and in a given location
func (vmf *VirtualMachineFactory) createResourceGroup(ctx context.Context, name, location string) armresources.ResourceGroup {
	param := armresources.ResourceGroup{
		Location: to.Ptr(location),
	}

	fmt.Printf("Building an Azure Resource Group named %q...\n", name)
	res, err := vmf.groupsClient.CreateOrUpdate(ctx, name, param, nil)
	HandleErr(err)
	return res.ResourceGroup
}

// createVirtualNetwork creates an Azure Virtual Network with a 10.0.0.0/16 CIDR with a 10.0.0.0/24 subnet
func (vmf *VirtualMachineFactory) createVirtualNetwork(ctx context.Context, vmStack *VirtualMachineStack) armnetwork.VirtualNetwork {
	param := armnetwork.VirtualNetwork{
		Location: to.Ptr(vmStack.Location),
		Name:     to.Ptr(vmStack.name + "-vnet"),
		Properties: &armnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{to.Ptr("10.0.0.0/16")},
			},
			Subnets: []*armnetwork.Subnet{
				{
					Name: to.Ptr("subnet1"),
					Properties: &armnetwork.SubnetPropertiesFormat{
						AddressPrefix:        to.Ptr("10.0.0.0/24"),
						NetworkSecurityGroup: &vmStack.SecurityGroup,
					},
				},
			},
		},
	}

	fmt.Printf("Building an Azure Virtual Network named %q...\n", *param.Name)
	poller, err := vmf.vnetClient.BeginCreateOrUpdate(ctx, vmStack.name, *param.Name, param, nil)
	HandleErr(err)
	res := HandleErrPoller(ctx, poller)
	return res.VirtualNetwork
}

// createSecurityGroup creates an Azure Network Security Group to allow SSH on port 22
func (vmf *VirtualMachineFactory) createSecurityGroup(ctx context.Context, name, location string) armnetwork.SecurityGroup {
	param := armnetwork.SecurityGroup{
		Location: to.Ptr(location),
		Name:     to.Ptr(name + "-nsg"),
		Properties: &armnetwork.SecurityGroupPropertiesFormat{
			SecurityRules: []*armnetwork.SecurityRule{
				{
					Name: to.Ptr("ssh"),
					Properties: &armnetwork.SecurityRulePropertiesFormat{
						Access:                   to.Ptr(armnetwork.SecurityRuleAccessAllow),
						Direction:                to.Ptr(armnetwork.SecurityRuleDirectionInbound),
						Protocol:                 to.Ptr(armnetwork.SecurityRuleProtocolAsterisk),
						Description:              to.Ptr("allow ssh on 22"),
						DestinationAddressPrefix: to.Ptr("*"),
						DestinationPortRange:     to.Ptr("22"),
						Priority:                 to.Ptr(int32(101)),
						SourcePortRange:          to.Ptr("*"),
						SourceAddressPrefix:      to.Ptr("*"),
					},
				},
			},
		},
	}

	fmt.Printf("Building an Azure Network Security Group named %q...\n", *param.Name)
	poller, err := vmf.nsgClient.BeginCreateOrUpdate(ctx, name, *param.Name, param, nil)
	HandleErr(err)
	res := HandleErrPoller(ctx, poller)
	return res.SecurityGroup
}

// createVirtualMachine creates an Azure Virtual Machine
func (vmf *VirtualMachineFactory) createVirtualMachine(ctx context.Context, vmStack *VirtualMachineStack) armcompute.VirtualMachine {
	param := linuxVM(vmStack)

	fmt.Printf("Building an Azure Virtual Machine named %q...\n", *param.Name)
	poller, err := vmf.vmClient.BeginCreateOrUpdate(ctx, vmStack.name, *param.Name, param, nil)
	HandleErr(err)
	res := HandleErrPoller(ctx, poller)
	return res.VirtualMachine
}

// getFirstNetworkInterface returns the first network interface on the vmStack Virtual Machine
func (vmf *VirtualMachineFactory) getFirstNetworkInterface(ctx context.Context, vmStack *VirtualMachineStack) armnetwork.Interface {
	iface := vmStack.VirtualMachine.Properties.NetworkProfile.NetworkInterfaces[0]
	parsed := HandleErrWithResult(arm.ParseResourceID(*iface.ID))
	fmt.Printf("Fetching the first Network Interface named %q connected to the VM...\n", parsed.Name)
	res := HandleErrWithResult(vmf.nicClient.Get(ctx, vmStack.name, parsed.Name, nil))
	return res.Interface
}

// getFirstNetworkInterface returns the first network interface on the vmStack Virtual Machine
func (vmf *VirtualMachineFactory) getPublicIPAddress(ctx context.Context, vmStack *VirtualMachineStack) armnetwork.PublicIPAddress {
	pipName := vmStack.NetworkInterface.Properties.IPConfigurations[0].Properties.PublicIPAddress.Name
	fmt.Printf("Fetching the Public IP Address named %q connected to the VM...\n", *pipName)
	res := HandleErrWithResult(vmf.pipClient.Get(ctx, vmStack.name, *pipName, nil))
	return res.PublicIPAddress
}

// linuxVM builds a Linux Virtual Machine structure
func linuxVM(vmStack *VirtualMachineStack) armcompute.VirtualMachine {
	return armcompute.VirtualMachine{
		Location: to.Ptr(vmStack.Location),
		Name:     to.Ptr(vmStack.name + "-vm"),
		Properties: &armcompute.VirtualMachineProperties{
			HardwareProfile: &armcompute.HardwareProfile{
				VMSize: to.Ptr(armcompute.VirtualMachineSizeTypesStandardD2SV3),
			},
			NetworkProfile: networkProfile(vmStack),
			OSProfile:      linuxOSProfile(vmStack),
			StorageProfile: &armcompute.StorageProfile{
				ImageReference: &armcompute.ImageReference{
					Publisher: to.Ptr("Canonical"),
					Offer:     to.Ptr("UbuntuServer"),
					SKU:       to.Ptr("18.04-LTS"),
					Version:   to.Ptr("latest"),
				},
			},
		},
	}
}

// networkProfile builds a Virtual Machine network profile requesting a public IP
func networkProfile(vmStack *VirtualMachineStack) *armcompute.NetworkProfile {
	firstSubnet := vmStack.VirtualNetwork.Properties.Subnets[0]
	return &armcompute.NetworkProfile{
		NetworkAPIVersion: to.Ptr(armcompute.NetworkAPIVersionTwoThousandTwenty1101),
		NetworkInterfaceConfigurations: []*armcompute.VirtualMachineNetworkInterfaceConfiguration{
			{
				Name: to.Ptr(vmStack.name + "-nic"),
				Properties: &armcompute.VirtualMachineNetworkInterfaceConfigurationProperties{
					IPConfigurations: []*armcompute.VirtualMachineNetworkInterfaceIPConfiguration{
						{
							Name: to.Ptr(vmStack.name + "-nic-conf"),
							Properties: &armcompute.VirtualMachineNetworkInterfaceIPConfigurationProperties{
								Primary: to.Ptr(true),
								Subnet: &armcompute.SubResource{
									ID: firstSubnet.ID,
								},
								PublicIPAddressConfiguration: &armcompute.VirtualMachinePublicIPAddressConfiguration{
									Name: to.Ptr(vmStack.name + "-pip"),
									Properties: &armcompute.VirtualMachinePublicIPAddressConfigurationProperties{
										PublicIPAllocationMethod: to.Ptr(armcompute.PublicIPAllocationMethodStatic),
										PublicIPAddressVersion:   to.Ptr(armcompute.IPVersionsIPv4),
									},
								},
							},
						},
					},
					Primary: to.Ptr(true),
				},
			},
		},
	}
}

// linuxOSProfile creates an Azure VM OS profile with only SSH access for the devops admin user
func linuxOSProfile(vmStack *VirtualMachineStack) *armcompute.OSProfile {
	sshKeyData := HandleErrWithResult(ioutil.ReadFile(vmStack.sshKeyPath))
	cloudInitContent := HandleErrWithResult(ioutil.ReadFile("./cloud-init/init.yml"))
	b64EncodedInitScript := base64.StdEncoding.EncodeToString(cloudInitContent)
	return &armcompute.OSProfile{
		AdminUsername: to.Ptr("devops"),
		ComputerName:  to.Ptr(vmStack.name),
		CustomData:    to.Ptr(b64EncodedInitScript),
		LinuxConfiguration: &armcompute.LinuxConfiguration{
			DisablePasswordAuthentication: to.Ptr(true),
			SSH: &armcompute.SSHConfiguration{
				PublicKeys: []*armcompute.SSHPublicKey{
					{
						Path:    to.Ptr("/home/devops/.ssh/authorized_keys"),
						KeyData: to.Ptr(string(sshKeyData)),
					},
				},
			},
		},
	}
}
