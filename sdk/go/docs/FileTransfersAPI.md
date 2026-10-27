# \FileTransfersAPI

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateClipboardAudit**](FileTransfersAPI.md#CreateClipboardAudit) | **Post** /v1/me/clipboard-audits | Report a controller-observed clipboard sync for auditing
[**CreateFileTransferAudit**](FileTransfersAPI.md#CreateFileTransferAudit) | **Post** /v1/me/file-transfer-audits | Report a controller-observed file transfer for auditing
[**ListMyAuditLogs**](FileTransfersAPI.md#ListMyAuditLogs) | **Get** /v1/me/audit-logs | Read back the data-channel audit trail that belongs to the caller



## CreateClipboardAudit

> CreateClipboardAudit(ctx).ClipboardAuditInput(clipboardAuditInput).Execute()

Report a controller-observed clipboard sync for auditing



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
	clipboardAuditInput := *openapiclient.NewClipboardAuditInput("DeviceId_example", "TransferId_example", "Direction_example", "Status_example", "MimeType_example") // ClipboardAuditInput | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.FileTransfersAPI.CreateClipboardAudit(context.Background()).ClipboardAuditInput(clipboardAuditInput).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FileTransfersAPI.CreateClipboardAudit``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiCreateClipboardAuditRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **clipboardAuditInput** | [**ClipboardAuditInput**](ClipboardAuditInput.md) |  | 

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


## ListMyAuditLogs

> OwnerAuditLogPage ListMyAuditLogs(ctx).Action(action).Cursor(cursor).Limit(limit).Execute()

Read back the data-channel audit trail that belongs to the caller



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
	action := "action_example" // string | Narrow to one audit action. An unsupported value is a 400. The aggregate names return one row per transfer; the direction-scoped names return the rows that went that way.  (optional)
	cursor := "cursor_example" // string |  (optional)
	limit := int32(56) // int32 |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FileTransfersAPI.ListMyAuditLogs(context.Background()).Action(action).Cursor(cursor).Limit(limit).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FileTransfersAPI.ListMyAuditLogs``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListMyAuditLogs`: OwnerAuditLogPage
	fmt.Fprintf(os.Stdout, "Response from `FileTransfersAPI.ListMyAuditLogs`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiListMyAuditLogsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **action** | **string** | Narrow to one audit action. An unsupported value is a 400. The aggregate names return one row per transfer; the direction-scoped names return the rows that went that way.  | 
 **cursor** | **string** |  | 
 **limit** | **int32** |  | 

### Return type

[**OwnerAuditLogPage**](OwnerAuditLogPage.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

