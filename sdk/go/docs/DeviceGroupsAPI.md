# \DeviceGroupsAPI

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateMyDeviceGroup**](DeviceGroupsAPI.md#CreateMyDeviceGroup) | **Post** /v1/me/device-groups | Create a private address-book group
[**DeleteMyDeviceGroup**](DeviceGroupsAPI.md#DeleteMyDeviceGroup) | **Delete** /v1/me/device-groups/{group_id} | Delete one of the caller&#39;s address-book groups
[**ListMyDeviceGroups**](DeviceGroupsAPI.md#ListMyDeviceGroups) | **Get** /v1/me/device-groups | The caller&#39;s private address-book groups
[**ReplaceMyDeviceGroupDevices**](DeviceGroupsAPI.md#ReplaceMyDeviceGroupDevices) | **Put** /v1/me/device-groups/{group_id}/devices | Replace the devices in one of the caller&#39;s groups
[**UpdateMyDeviceGroup**](DeviceGroupsAPI.md#UpdateMyDeviceGroup) | **Patch** /v1/me/device-groups/{group_id} | Update one of the caller&#39;s address-book groups



## CreateMyDeviceGroup

> UserDeviceGroup CreateMyDeviceGroup(ctx).UserDeviceGroupInput(userDeviceGroupInput).Execute()

Create a private address-book group

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
	userDeviceGroupInput := *openapiclient.NewUserDeviceGroupInput("Name_example") // UserDeviceGroupInput | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DeviceGroupsAPI.CreateMyDeviceGroup(context.Background()).UserDeviceGroupInput(userDeviceGroupInput).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DeviceGroupsAPI.CreateMyDeviceGroup``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateMyDeviceGroup`: UserDeviceGroup
	fmt.Fprintf(os.Stdout, "Response from `DeviceGroupsAPI.CreateMyDeviceGroup`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiCreateMyDeviceGroupRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **userDeviceGroupInput** | [**UserDeviceGroupInput**](UserDeviceGroupInput.md) |  | 

### Return type

[**UserDeviceGroup**](UserDeviceGroup.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteMyDeviceGroup

> DeleteMyDeviceGroup200Response DeleteMyDeviceGroup(ctx, groupId).Execute()

Delete one of the caller's address-book groups

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
	groupId := int64(789) // int64 | The numeric id of one of the caller's private device groups.

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DeviceGroupsAPI.DeleteMyDeviceGroup(context.Background(), groupId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DeviceGroupsAPI.DeleteMyDeviceGroup``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `DeleteMyDeviceGroup`: DeleteMyDeviceGroup200Response
	fmt.Fprintf(os.Stdout, "Response from `DeviceGroupsAPI.DeleteMyDeviceGroup`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**groupId** | **int64** | The numeric id of one of the caller&#39;s private device groups. | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteMyDeviceGroupRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**DeleteMyDeviceGroup200Response**](DeleteMyDeviceGroup200Response.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListMyDeviceGroups

> UserDeviceGroupList ListMyDeviceGroups(ctx).Execute()

The caller's private address-book groups



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
	resp, r, err := apiClient.DeviceGroupsAPI.ListMyDeviceGroups(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DeviceGroupsAPI.ListMyDeviceGroups``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListMyDeviceGroups`: UserDeviceGroupList
	fmt.Fprintf(os.Stdout, "Response from `DeviceGroupsAPI.ListMyDeviceGroups`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiListMyDeviceGroupsRequest struct via the builder pattern


### Return type

[**UserDeviceGroupList**](UserDeviceGroupList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ReplaceMyDeviceGroupDevices

> UserDeviceGroup ReplaceMyDeviceGroupDevices(ctx, groupId).UserDeviceGroupDevicesInput(userDeviceGroupDevicesInput).Execute()

Replace the devices in one of the caller's groups



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
	groupId := int64(789) // int64 | The numeric id of one of the caller's private device groups.
	userDeviceGroupDevicesInput := *openapiclient.NewUserDeviceGroupDevicesInput([]string{"DeviceIds_example"}) // UserDeviceGroupDevicesInput | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DeviceGroupsAPI.ReplaceMyDeviceGroupDevices(context.Background(), groupId).UserDeviceGroupDevicesInput(userDeviceGroupDevicesInput).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DeviceGroupsAPI.ReplaceMyDeviceGroupDevices``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ReplaceMyDeviceGroupDevices`: UserDeviceGroup
	fmt.Fprintf(os.Stdout, "Response from `DeviceGroupsAPI.ReplaceMyDeviceGroupDevices`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**groupId** | **int64** | The numeric id of one of the caller&#39;s private device groups. | 

### Other Parameters

Other parameters are passed through a pointer to a apiReplaceMyDeviceGroupDevicesRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **userDeviceGroupDevicesInput** | [**UserDeviceGroupDevicesInput**](UserDeviceGroupDevicesInput.md) |  | 

### Return type

[**UserDeviceGroup**](UserDeviceGroup.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UpdateMyDeviceGroup

> UserDeviceGroup UpdateMyDeviceGroup(ctx, groupId).UserDeviceGroupPatch(userDeviceGroupPatch).Execute()

Update one of the caller's address-book groups

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
	groupId := int64(789) // int64 | The numeric id of one of the caller's private device groups.
	userDeviceGroupPatch := *openapiclient.NewUserDeviceGroupPatch() // UserDeviceGroupPatch | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DeviceGroupsAPI.UpdateMyDeviceGroup(context.Background(), groupId).UserDeviceGroupPatch(userDeviceGroupPatch).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DeviceGroupsAPI.UpdateMyDeviceGroup``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `UpdateMyDeviceGroup`: UserDeviceGroup
	fmt.Fprintf(os.Stdout, "Response from `DeviceGroupsAPI.UpdateMyDeviceGroup`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**groupId** | **int64** | The numeric id of one of the caller&#39;s private device groups. | 

### Other Parameters

Other parameters are passed through a pointer to a apiUpdateMyDeviceGroupRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **userDeviceGroupPatch** | [**UserDeviceGroupPatch**](UserDeviceGroupPatch.md) |  | 

### Return type

[**UserDeviceGroup**](UserDeviceGroup.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

