package helpers

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armruntime "github.com/Azure/azure-sdk-for-go/sdk/azcore/arm/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

const (
	DefaultPollingFreq = 10 * time.Second
)

type ClientBuilderFunc[T any] func(string, azcore.TokenCredential, *arm.ClientOptions) (*T, error)

func MustGetenv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("please add your %s to the .env file", key)
	}
	return val
}

func BuildClient[T any](subID string, cred *azidentity.DefaultAzureCredential, builderFunc ClientBuilderFunc[T]) *T {
	return HandleErrWithResult(builderFunc(subID, cred, nil))
}

func HandleErrPoller[T any](ctx context.Context, poller *armruntime.Poller[T]) T {
	res, err := poller.PollUntilDone(ctx, DefaultPollingFreq)
	HandleErr(err)
	return res
}

func HandleErrWithResult[T any](result T, err error) T {
	HandleErr(err)
	return result
}

func HandleErr(err error) {
	if err != nil {
		panic(err)
	}
}
