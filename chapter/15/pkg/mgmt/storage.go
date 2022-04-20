package mgmt

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	. "github.com/PacktPublishing/Go-for-DevOps/chapter/15/pkg/helpers"
)

type StorageStack struct {
	Location      string
	name          string
	Cred          azcore.TokenCredential
	ResourceGroup armresources.ResourceGroup
	Account       armstorage.Account
	AccountKey    *armstorage.AccountKey
}

type StorageFactory struct {
	subscriptionID string
	cred           azcore.TokenCredential
	groupsClient   *armresources.ResourceGroupsClient
	storageClient  *armstorage.AccountsClient
}

// NewStorageFactory instantiates an Azure Storage factory for building an Azure Storage playground
func NewStorageFactory(subscriptionID string) *StorageFactory {
	cred := HandleErrWithResult(azidentity.NewDefaultAzureCredential(nil))
	return &StorageFactory{
		cred:           cred,
		subscriptionID: subscriptionID,
		groupsClient:   BuildClient(subscriptionID, cred, armresources.NewResourceGroupsClient),
		storageClient:  BuildClient(subscriptionID, cred, armstorage.NewAccountsClient),
	}
}

func (sf *StorageFactory) CreateStorageStack(ctx context.Context, location string) *StorageStack {
	stack := &StorageStack{
		name: haiku.Haikunate(),
	}
	stack.ResourceGroup = sf.createResourceGroup(ctx, stack.name, location)
	stack.Account = sf.createStorageAccount(ctx, stack.name, location)
	stack.AccountKey = sf.getPrimaryAccountKey(ctx, stack)
	return stack
}

func (sf *StorageFactory) DestroyStorageStack(ctx context.Context, stack *StorageStack) {
	_, err := sf.groupsClient.BeginDelete(ctx, stack.name, nil)
	HandleErr(err)
}

// createResourceGroup creates an Azure resource by name and in a given location
func (sf *StorageFactory) createResourceGroup(ctx context.Context, name, location string) armresources.ResourceGroup {
	param := armresources.ResourceGroup{
		Location: to.Ptr(location),
	}

	fmt.Printf("Building an Azure Resource Group named %q...\n", name)
	res, err := sf.groupsClient.CreateOrUpdate(ctx, name, param, nil)
	HandleErr(err)
	return res.ResourceGroup
}

// createStorageAccount creates an Azure Storage Account
func (sf *StorageFactory) createStorageAccount(ctx context.Context, name, location string) armstorage.Account {
	param := armstorage.AccountCreateParameters{
		Location: to.Ptr(location),
		Kind:     to.Ptr(armstorage.KindBlockBlobStorage),
		SKU: &armstorage.SKU{
			Name: to.Ptr(armstorage.SKUNamePremiumLRS),
			Tier: to.Ptr(armstorage.SKUTierPremium),
		},
	}

	accountName := strings.Replace(name, "-", "", -1)
	fmt.Printf("Building an Azure Storage Account named %q...\n", accountName)
	poller, err := sf.storageClient.BeginCreate(ctx, name, accountName, param, nil)
	HandleErr(err)
	res := HandleErrPoller(ctx, poller)
	return res.Account
}

func (sf *StorageFactory) getPrimaryAccountKey(ctx context.Context, stack *StorageStack) *armstorage.AccountKey {
	fmt.Printf("Fetching the Azure Storage Account shared key...\n")
	res, err := sf.storageClient.ListKeys(ctx, stack.name, *stack.Account.Name, nil)
	HandleErr(err)
	return res.Keys[0]
}

func (ss *StorageStack) ServiceClient() *azblob.ServiceClient {
	cred := HandleErrWithResult(azblob.NewSharedKeyCredential(*ss.Account.Name, *ss.AccountKey.Value))
	blobURI := *ss.Account.Properties.PrimaryEndpoints.Blob
	client, err := azblob.NewServiceClientWithSharedKey(blobURI, cred, nil)
	HandleErr(err)
	return client
}
