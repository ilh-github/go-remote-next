# QuickDesk.Signaling.Client.Api.DeviceGroupsApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
|--------|--------------|-------------|
| [**CreateMyDeviceGroup**](DeviceGroupsApi.md#createmydevicegroup) | **POST** /v1/me/device-groups | Create a private address-book group |
| [**DeleteMyDeviceGroup**](DeviceGroupsApi.md#deletemydevicegroup) | **DELETE** /v1/me/device-groups/{group_id} | Delete one of the caller&#39;s address-book groups |
| [**ListMyDeviceGroups**](DeviceGroupsApi.md#listmydevicegroups) | **GET** /v1/me/device-groups | The caller&#39;s private address-book groups |
| [**ReplaceMyDeviceGroupDevices**](DeviceGroupsApi.md#replacemydevicegroupdevices) | **PUT** /v1/me/device-groups/{group_id}/devices | Replace the devices in one of the caller&#39;s groups |
| [**UpdateMyDeviceGroup**](DeviceGroupsApi.md#updatemydevicegroup) | **PATCH** /v1/me/device-groups/{group_id} | Update one of the caller&#39;s address-book groups |

<a id="createmydevicegroup"></a>
# **CreateMyDeviceGroup**
> UserDeviceGroup CreateMyDeviceGroup (UserDeviceGroupInput userDeviceGroupInput)

Create a private address-book group

### Example
```csharp
using System.Collections.Generic;
using System.Diagnostics;
using System.Net.Http;
using QuickDesk.Signaling.Client.Api;
using QuickDesk.Signaling.Client.Client;
using QuickDesk.Signaling.Client.Model;

namespace Example
{
    public class CreateMyDeviceGroupExample
    {
        public static void Main()
        {
            Configuration config = new Configuration();
            config.BasePath = "https://your-signaling-host";
            // Configure API key authorization: ApiKeyAuth
            config.AddApiKey("X-API-Key", "YOUR_API_KEY");
            // Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
            // config.AddApiKeyPrefix("X-API-Key", "Bearer");
            // Configure Bearer token for authorization: BearerAuth
            config.AccessToken = "YOUR_BEARER_TOKEN";

            // create instances of HttpClient, HttpClientHandler to be reused later with different Api classes
            HttpClient httpClient = new HttpClient();
            HttpClientHandler httpClientHandler = new HttpClientHandler();
            var apiInstance = new DeviceGroupsApi(httpClient, config, httpClientHandler);
            var userDeviceGroupInput = new UserDeviceGroupInput(); // UserDeviceGroupInput | 

            try
            {
                // Create a private address-book group
                UserDeviceGroup result = apiInstance.CreateMyDeviceGroup(userDeviceGroupInput);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling DeviceGroupsApi.CreateMyDeviceGroup: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the CreateMyDeviceGroupWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Create a private address-book group
    ApiResponse<UserDeviceGroup> response = apiInstance.CreateMyDeviceGroupWithHttpInfo(userDeviceGroupInput);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling DeviceGroupsApi.CreateMyDeviceGroupWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **userDeviceGroupInput** | [**UserDeviceGroupInput**](UserDeviceGroupInput.md) |  |  |

### Return type

[**UserDeviceGroup**](UserDeviceGroup.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **201** | The created group. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **409** | The caller already has a group with that name. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="deletemydevicegroup"></a>
# **DeleteMyDeviceGroup**
> DeleteMyDeviceGroup200Response DeleteMyDeviceGroup (long groupId)

Delete one of the caller's address-book groups

### Example
```csharp
using System.Collections.Generic;
using System.Diagnostics;
using System.Net.Http;
using QuickDesk.Signaling.Client.Api;
using QuickDesk.Signaling.Client.Client;
using QuickDesk.Signaling.Client.Model;

namespace Example
{
    public class DeleteMyDeviceGroupExample
    {
        public static void Main()
        {
            Configuration config = new Configuration();
            config.BasePath = "https://your-signaling-host";
            // Configure API key authorization: ApiKeyAuth
            config.AddApiKey("X-API-Key", "YOUR_API_KEY");
            // Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
            // config.AddApiKeyPrefix("X-API-Key", "Bearer");
            // Configure Bearer token for authorization: BearerAuth
            config.AccessToken = "YOUR_BEARER_TOKEN";

            // create instances of HttpClient, HttpClientHandler to be reused later with different Api classes
            HttpClient httpClient = new HttpClient();
            HttpClientHandler httpClientHandler = new HttpClientHandler();
            var apiInstance = new DeviceGroupsApi(httpClient, config, httpClientHandler);
            var groupId = 789L;  // long | The numeric id of one of the caller's private device groups.

            try
            {
                // Delete one of the caller's address-book groups
                DeleteMyDeviceGroup200Response result = apiInstance.DeleteMyDeviceGroup(groupId);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling DeviceGroupsApi.DeleteMyDeviceGroup: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the DeleteMyDeviceGroupWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Delete one of the caller's address-book groups
    ApiResponse<DeleteMyDeviceGroup200Response> response = apiInstance.DeleteMyDeviceGroupWithHttpInfo(groupId);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling DeviceGroupsApi.DeleteMyDeviceGroupWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **groupId** | **long** | The numeric id of one of the caller&#39;s private device groups. |  |

### Return type

[**DeleteMyDeviceGroup200Response**](DeleteMyDeviceGroup200Response.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The group was deleted. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="listmydevicegroups"></a>
# **ListMyDeviceGroups**
> UserDeviceGroupList ListMyDeviceGroups ()

The caller's private address-book groups

Owner-scoped groups are separate from the administrator-wide device inventory under `/v1/admin/groups`. Each returned group carries its current `device_ids`, so a client can render filters and membership without issuing one request per group. 

### Example
```csharp
using System.Collections.Generic;
using System.Diagnostics;
using System.Net.Http;
using QuickDesk.Signaling.Client.Api;
using QuickDesk.Signaling.Client.Client;
using QuickDesk.Signaling.Client.Model;

namespace Example
{
    public class ListMyDeviceGroupsExample
    {
        public static void Main()
        {
            Configuration config = new Configuration();
            config.BasePath = "https://your-signaling-host";
            // Configure API key authorization: ApiKeyAuth
            config.AddApiKey("X-API-Key", "YOUR_API_KEY");
            // Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
            // config.AddApiKeyPrefix("X-API-Key", "Bearer");
            // Configure Bearer token for authorization: BearerAuth
            config.AccessToken = "YOUR_BEARER_TOKEN";

            // create instances of HttpClient, HttpClientHandler to be reused later with different Api classes
            HttpClient httpClient = new HttpClient();
            HttpClientHandler httpClientHandler = new HttpClientHandler();
            var apiInstance = new DeviceGroupsApi(httpClient, config, httpClientHandler);

            try
            {
                // The caller's private address-book groups
                UserDeviceGroupList result = apiInstance.ListMyDeviceGroups();
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling DeviceGroupsApi.ListMyDeviceGroups: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the ListMyDeviceGroupsWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // The caller's private address-book groups
    ApiResponse<UserDeviceGroupList> response = apiInstance.ListMyDeviceGroupsWithHttpInfo();
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling DeviceGroupsApi.ListMyDeviceGroupsWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters
This endpoint does not need any parameter.
### Return type

[**UserDeviceGroupList**](UserDeviceGroupList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The caller&#39;s groups. |  -  |
| **401** | Missing or expired credentials. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="replacemydevicegroupdevices"></a>
# **ReplaceMyDeviceGroupDevices**
> UserDeviceGroup ReplaceMyDeviceGroupDevices (long groupId, UserDeviceGroupDevicesInput userDeviceGroupDevicesInput)

Replace the devices in one of the caller's groups

Replaces the whole membership atomically. Every device must currently belong to the caller; an unknown or foreign device is rejected with `403 DEVICE_NOT_OWNED`. Sending an empty `device_ids` array clears the group. 

### Example
```csharp
using System.Collections.Generic;
using System.Diagnostics;
using System.Net.Http;
using QuickDesk.Signaling.Client.Api;
using QuickDesk.Signaling.Client.Client;
using QuickDesk.Signaling.Client.Model;

namespace Example
{
    public class ReplaceMyDeviceGroupDevicesExample
    {
        public static void Main()
        {
            Configuration config = new Configuration();
            config.BasePath = "https://your-signaling-host";
            // Configure API key authorization: ApiKeyAuth
            config.AddApiKey("X-API-Key", "YOUR_API_KEY");
            // Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
            // config.AddApiKeyPrefix("X-API-Key", "Bearer");
            // Configure Bearer token for authorization: BearerAuth
            config.AccessToken = "YOUR_BEARER_TOKEN";

            // create instances of HttpClient, HttpClientHandler to be reused later with different Api classes
            HttpClient httpClient = new HttpClient();
            HttpClientHandler httpClientHandler = new HttpClientHandler();
            var apiInstance = new DeviceGroupsApi(httpClient, config, httpClientHandler);
            var groupId = 789L;  // long | The numeric id of one of the caller's private device groups.
            var userDeviceGroupDevicesInput = new UserDeviceGroupDevicesInput(); // UserDeviceGroupDevicesInput | 

            try
            {
                // Replace the devices in one of the caller's groups
                UserDeviceGroup result = apiInstance.ReplaceMyDeviceGroupDevices(groupId, userDeviceGroupDevicesInput);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling DeviceGroupsApi.ReplaceMyDeviceGroupDevices: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the ReplaceMyDeviceGroupDevicesWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Replace the devices in one of the caller's groups
    ApiResponse<UserDeviceGroup> response = apiInstance.ReplaceMyDeviceGroupDevicesWithHttpInfo(groupId, userDeviceGroupDevicesInput);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling DeviceGroupsApi.ReplaceMyDeviceGroupDevicesWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **groupId** | **long** | The numeric id of one of the caller&#39;s private device groups. |  |
| **userDeviceGroupDevicesInput** | [**UserDeviceGroupDevicesInput**](UserDeviceGroupDevicesInput.md) |  |  |

### Return type

[**UserDeviceGroup**](UserDeviceGroup.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The updated group. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="updatemydevicegroup"></a>
# **UpdateMyDeviceGroup**
> UserDeviceGroup UpdateMyDeviceGroup (long groupId, UserDeviceGroupPatch userDeviceGroupPatch)

Update one of the caller's address-book groups

### Example
```csharp
using System.Collections.Generic;
using System.Diagnostics;
using System.Net.Http;
using QuickDesk.Signaling.Client.Api;
using QuickDesk.Signaling.Client.Client;
using QuickDesk.Signaling.Client.Model;

namespace Example
{
    public class UpdateMyDeviceGroupExample
    {
        public static void Main()
        {
            Configuration config = new Configuration();
            config.BasePath = "https://your-signaling-host";
            // Configure API key authorization: ApiKeyAuth
            config.AddApiKey("X-API-Key", "YOUR_API_KEY");
            // Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
            // config.AddApiKeyPrefix("X-API-Key", "Bearer");
            // Configure Bearer token for authorization: BearerAuth
            config.AccessToken = "YOUR_BEARER_TOKEN";

            // create instances of HttpClient, HttpClientHandler to be reused later with different Api classes
            HttpClient httpClient = new HttpClient();
            HttpClientHandler httpClientHandler = new HttpClientHandler();
            var apiInstance = new DeviceGroupsApi(httpClient, config, httpClientHandler);
            var groupId = 789L;  // long | The numeric id of one of the caller's private device groups.
            var userDeviceGroupPatch = new UserDeviceGroupPatch(); // UserDeviceGroupPatch | 

            try
            {
                // Update one of the caller's address-book groups
                UserDeviceGroup result = apiInstance.UpdateMyDeviceGroup(groupId, userDeviceGroupPatch);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling DeviceGroupsApi.UpdateMyDeviceGroup: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the UpdateMyDeviceGroupWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Update one of the caller's address-book groups
    ApiResponse<UserDeviceGroup> response = apiInstance.UpdateMyDeviceGroupWithHttpInfo(groupId, userDeviceGroupPatch);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling DeviceGroupsApi.UpdateMyDeviceGroupWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **groupId** | **long** | The numeric id of one of the caller&#39;s private device groups. |  |
| **userDeviceGroupPatch** | [**UserDeviceGroupPatch**](UserDeviceGroupPatch.md) |  |  |

### Return type

[**UserDeviceGroup**](UserDeviceGroup.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The updated group. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |
| **409** | The caller already has a group with that name. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

