# \NotificationsAPI

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**DeleteMetricAlert**](NotificationsAPI.md#DeleteMetricAlert) | **Delete** /v1/me/devices/{device_id}/metric-alert | Delete the owner&#39;s metric alert rule
[**GetMetricAlert**](NotificationsAPI.md#GetMetricAlert) | **Get** /v1/me/devices/{device_id}/metric-alert | The owner&#39;s CPU, memory, and disk alert rule
[**GetOfflineAlert**](NotificationsAPI.md#GetOfflineAlert) | **Get** /v1/me/devices/{device_id}/offline-alert | The owner&#39;s offline-alert rule for a device
[**ListNotificationChannels**](NotificationsAPI.md#ListNotificationChannels) | **Get** /v1/me/notification-channels | The user&#39;s notification channels
[**PutMetricAlert**](NotificationsAPI.md#PutMetricAlert) | **Put** /v1/me/devices/{device_id}/metric-alert | Create or replace the owner&#39;s metric alert rule
[**PutOfflineAlert**](NotificationsAPI.md#PutOfflineAlert) | **Put** /v1/me/devices/{device_id}/offline-alert | Create or replace the owner&#39;s offline-alert rule



## DeleteMetricAlert

> DeleteMetricAlert(ctx, deviceId).Execute()

Delete the owner's metric alert rule

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
	deviceId := "deviceId_example" // string | The device id shown on the controlled machine.

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.NotificationsAPI.DeleteMetricAlert(context.Background(), deviceId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `NotificationsAPI.DeleteMetricAlert``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**deviceId** | **string** | The device id shown on the controlled machine. | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteMetricAlertRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

 (empty response body)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetMetricAlert

> MetricAlertRule GetMetricAlert(ctx, deviceId).Execute()

The owner's CPU, memory, and disk alert rule



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
	deviceId := "deviceId_example" // string | The device id shown on the controlled machine.

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.NotificationsAPI.GetMetricAlert(context.Background(), deviceId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `NotificationsAPI.GetMetricAlert``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetMetricAlert`: MetricAlertRule
	fmt.Fprintf(os.Stdout, "Response from `NotificationsAPI.GetMetricAlert`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**deviceId** | **string** | The device id shown on the controlled machine. | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetMetricAlertRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**MetricAlertRule**](MetricAlertRule.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetOfflineAlert

> OfflineAlertRule GetOfflineAlert(ctx, deviceId).Execute()

The owner's offline-alert rule for a device



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
	deviceId := "deviceId_example" // string | The device id shown on the controlled machine.

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.NotificationsAPI.GetOfflineAlert(context.Background(), deviceId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `NotificationsAPI.GetOfflineAlert``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetOfflineAlert`: OfflineAlertRule
	fmt.Fprintf(os.Stdout, "Response from `NotificationsAPI.GetOfflineAlert`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**deviceId** | **string** | The device id shown on the controlled machine. | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetOfflineAlertRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**OfflineAlertRule**](OfflineAlertRule.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListNotificationChannels

> NotificationChannelList ListNotificationChannels(ctx).Execute()

The user's notification channels



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

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.NotificationsAPI.ListNotificationChannels(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `NotificationsAPI.ListNotificationChannels``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListNotificationChannels`: NotificationChannelList
	fmt.Fprintf(os.Stdout, "Response from `NotificationsAPI.ListNotificationChannels`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiListNotificationChannelsRequest struct via the builder pattern


### Return type

[**NotificationChannelList**](NotificationChannelList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PutMetricAlert

> MetricAlertRule PutMetricAlert(ctx, deviceId).MetricAlertInput(metricAlertInput).Execute()

Create or replace the owner's metric alert rule

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
	deviceId := "deviceId_example" // string | The device id shown on the controlled machine.
	metricAlertInput := *openapiclient.NewMetricAlertInput(false, int32(123), int32(123), int32(123), int32(123), int32(123)) // MetricAlertInput | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.NotificationsAPI.PutMetricAlert(context.Background(), deviceId).MetricAlertInput(metricAlertInput).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `NotificationsAPI.PutMetricAlert``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PutMetricAlert`: MetricAlertRule
	fmt.Fprintf(os.Stdout, "Response from `NotificationsAPI.PutMetricAlert`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**deviceId** | **string** | The device id shown on the controlled machine. | 

### Other Parameters

Other parameters are passed through a pointer to a apiPutMetricAlertRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **metricAlertInput** | [**MetricAlertInput**](MetricAlertInput.md) |  | 

### Return type

[**MetricAlertRule**](MetricAlertRule.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PutOfflineAlert

> OfflineAlertRule PutOfflineAlert(ctx, deviceId).OfflineAlertInput(offlineAlertInput).Execute()

Create or replace the owner's offline-alert rule

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
	deviceId := "deviceId_example" // string | The device id shown on the controlled machine.
	offlineAlertInput := *openapiclient.NewOfflineAlertInput(false) // OfflineAlertInput | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.NotificationsAPI.PutOfflineAlert(context.Background(), deviceId).OfflineAlertInput(offlineAlertInput).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `NotificationsAPI.PutOfflineAlert``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PutOfflineAlert`: OfflineAlertRule
	fmt.Fprintf(os.Stdout, "Response from `NotificationsAPI.PutOfflineAlert`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**deviceId** | **string** | The device id shown on the controlled machine. | 

### Other Parameters

Other parameters are passed through a pointer to a apiPutOfflineAlertRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **offlineAlertInput** | [**OfflineAlertInput**](OfflineAlertInput.md) |  | 

### Return type

[**OfflineAlertRule**](OfflineAlertRule.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

