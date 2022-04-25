package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/joho/godotenv"

	. "github.com/PacktPublishing/Go-for-DevOps/chapter/15/pkg/helpers"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/15/pkg/mgmt"
)

func init() {
	_ = godotenv.Load()
}

func main() {
	subscriptionID := MustGetenv("AZURE_SUBSCRIPTION_ID")
	factory := mgmt.NewStorageFactory(subscriptionID)
	fmt.Println("Staring to build Azure resources...")
	stack := factory.CreateStorageStack(context.Background(), "southcentralus")

	uploadBlobs(stack)
	printSASUris(stack)

	fmt.Println("Press enter to delete the infrastructure.")
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')
	factory.DestroyStorageStack(context.Background(), stack)
}

func uploadBlobs(stack *mgmt.StorageStack) {
	serviceClient := stack.ServiceClient()
	containerClient, err := serviceClient.NewContainerClient("jd-imgs")
	HandleErr(err)

	fmt.Printf("Creating a new container \"jd-imgs\" in the Storage Account...\n")
	_, err = containerClient.Create(context.Background(), nil)
	HandleErr(err)

	fmt.Printf("Reading all files ./blobs...\n")
	files, err := ioutil.ReadDir("./blobs")
	HandleErr(err)
	for _, file := range files {
		fmt.Printf("Uploading file %q to container jd-imgs...\n", file.Name())
		blobClient := HandleErrWithResult(containerClient.NewBlockBlobClient(file.Name()))
		osFile := HandleErrWithResult(os.Open(path.Join("./blobs", file.Name())))
		_ = HandleErrWithResult(blobClient.UploadFile(context.Background(), osFile, azblob.UploadOption{}))
	}
}

func printSASUris(stack *mgmt.StorageStack) {
	serviceClient := stack.ServiceClient()
	containerClient, err := serviceClient.NewContainerClient("jd-imgs")
	HandleErr(err)

	fmt.Printf("\nGenerating readonly links to blobs that expire in 2 hours...\n")
	files := HandleErrWithResult(ioutil.ReadDir("./blobs"))
	for _, file := range files {
		blobClient := HandleErrWithResult(containerClient.NewBlockBlobClient(file.Name()))
		permissions := azblob.BlobSASPermissions{
			Read: true,
		}
		now := time.Now().UTC()
		sasQuery := HandleErrWithResult(blobClient.GetSASToken(permissions, now, now.Add(2*time.Hour)))
		fmt.Println(blobClient.URL() + "?" + sasQuery.Encode())
	}
}
