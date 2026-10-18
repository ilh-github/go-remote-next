# \FileTransfersAPI

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateFileTransferAudit**](FileTransfersAPI.md#CreateFileTransferAudit) | **Post** /v1/me/file-transfer-audits | Report a controller-observed file transfer for auditing



## CreateFileTransferAudit

> CreateFileTransferAudit(ctx).FileTransferAuditInput(fileTransferAuditInput).Execute()

Report a controller-observed file transfer for auditing



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/ilh-github/go-remote-next/signaling"
)

func main() {
	fileTransferAuditInput := *openapiclient.NewFileTransferAuditInput("DeviceId_example", "TransferId_example", "Direction_example", "Status_example") // FileTransferAuditInput | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.FileTransfersAPI.CreateFileTransferAudit(context.Background()).FileTransferAuditInput(fileTransferAuditInput).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FileTransfersAPI.CreateFileTransferAudit``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiCreateFileTransferAuditRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **fileTransferAuditInput** | [**FileTransferAuditInput**](FileTransferAuditInput.md) |  | 

### Return type

 (empty response body)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

