package sdkclient

import "github.com/nobl9/nobl9-go/sdk"

type SdkClientProvider interface {
	GetClient() *sdk.Client
}
