# QuickDesk.Signaling.Client.Api.NotificationsApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
|--------|--------------|-------------|
| [**DeleteMetricAlert**](NotificationsApi.md#deletemetricalert) | **DELETE** /v1/me/devices/{device_id}/metric-alert | Delete the owner&#39;s metric alert rule |
| [**GetMetricAlert**](NotificationsApi.md#getmetricalert) | **GET** /v1/me/devices/{device_id}/metric-alert | The owner&#39;s CPU, memory, and disk alert rule |
| [**GetOfflineAlert**](NotificationsApi.md#getofflinealert) | **GET** /v1/me/devices/{device_id}/offline-alert | The owner&#39;s offline-alert rule for a device |
| [**ListNotificationChannels**](NotificationsApi.md#listnotificationchannels) | **GET** /v1/me/notification-channels | The user&#39;s notification channels |
| [**PutMetricAlert**](NotificationsApi.md#putmetricalert) | **PUT** /v1/me/devices/{device_id}/metric-alert | Create or replace the owner&#39;s metric alert rule |
| [**PutOfflineAlert**](NotificationsApi.md#putofflinealert) | **PUT** /v1/me/devices/{device_id}/offline-alert | Create or replace the owner&#39;s offline-alert rule |

<a id="deletemetricalert"></a>
# **DeleteMetricAlert**
> void DeleteMetricAlert (string deviceId)

Delete the owner's metric alert rule

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
    public class DeleteMetricAlertExample
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
            var apiInstance = new NotificationsApi(httpClient, config, httpClientHandler);
            var deviceId = "deviceId_example";  // string | The device id shown on the controlled machine.

            try
            {
                // Delete the owner's metric alert rule
                apiInstance.DeleteMetricAlert(deviceId);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling NotificationsApi.DeleteMetricAlert: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the DeleteMetricAlertWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Delete the owner's metric alert rule
    apiInstance.DeleteMetricAlertWithHttpInfo(deviceId);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling NotificationsApi.DeleteMetricAlertWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **deviceId** | **string** | The device id shown on the controlled machine. |  |

### Return type

void (empty response body)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **204** | The rule was deleted. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="getmetricalert"></a>
# **GetMetricAlert**
> MetricAlertRule GetMetricAlert (string deviceId)

The owner's CPU, memory, and disk alert rule

Returns safe defaults when the owner never configured a rule, along with the latest host sample and the independent episode state of each metric. Samples are one-per-device; this endpoint is not a time-series query. 

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
    public class GetMetricAlertExample
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
            var apiInstance = new NotificationsApi(httpClient, config, httpClientHandler);
            var deviceId = "deviceId_example";  // string | The device id shown on the controlled machine.

            try
            {
                // The owner's CPU, memory, and disk alert rule
                MetricAlertRule result = apiInstance.GetMetricAlert(deviceId);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling NotificationsApi.GetMetricAlert: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the GetMetricAlertWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // The owner's CPU, memory, and disk alert rule
    ApiResponse<MetricAlertRule> response = apiInstance.GetMetricAlertWithHttpInfo(deviceId);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling NotificationsApi.GetMetricAlertWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **deviceId** | **string** | The device id shown on the controlled machine. |  |

### Return type

[**MetricAlertRule**](MetricAlertRule.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The rule, latest sample, and metric episode state. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="getofflinealert"></a>
# **GetOfflineAlert**
> OfflineAlertRule GetOfflineAlert (string deviceId)

The owner's offline-alert rule for a device

Returns the documented defaults (`enabled: false`, `after_minutes: 10`) when the owner never configured a rule, so a client can render the form without a second \"does it exist\" call. 

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
    public class GetOfflineAlertExample
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
            var apiInstance = new NotificationsApi(httpClient, config, httpClientHandler);
            var deviceId = "deviceId_example";  // string | The device id shown on the controlled machine.

            try
            {
                // The owner's offline-alert rule for a device
                OfflineAlertRule result = apiInstance.GetOfflineAlert(deviceId);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling NotificationsApi.GetOfflineAlert: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the GetOfflineAlertWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // The owner's offline-alert rule for a device
    ApiResponse<OfflineAlertRule> response = apiInstance.GetOfflineAlertWithHttpInfo(deviceId);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling NotificationsApi.GetOfflineAlertWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **deviceId** | **string** | The device id shown on the controlled machine. |  |

### Return type

[**OfflineAlertRule**](OfflineAlertRule.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The rule, or its defaults. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="listnotificationchannels"></a>
# **ListNotificationChannels**
> NotificationChannelList ListNotificationChannels ()

The user's notification channels

`supported_events` is the authoritative list of event types a channel may subscribe to. The signing secret is never returned; deliveries are signed with `signature_header` using `signature_scheme`. 

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
    public class ListNotificationChannelsExample
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
            var apiInstance = new NotificationsApi(httpClient, config, httpClientHandler);

            try
            {
                // The user's notification channels
                NotificationChannelList result = apiInstance.ListNotificationChannels();
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling NotificationsApi.ListNotificationChannels: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the ListNotificationChannelsWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // The user's notification channels
    ApiResponse<NotificationChannelList> response = apiInstance.ListNotificationChannelsWithHttpInfo();
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling NotificationsApi.ListNotificationChannelsWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters
This endpoint does not need any parameter.
### Return type

[**NotificationChannelList**](NotificationChannelList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The caller&#39;s channels plus the subscription vocabulary. |  -  |
| **401** | Missing or expired credentials. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="putmetricalert"></a>
# **PutMetricAlert**
> MetricAlertRule PutMetricAlert (string deviceId, MetricAlertInput metricAlertInput)

Create or replace the owner's metric alert rule

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
    public class PutMetricAlertExample
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
            var apiInstance = new NotificationsApi(httpClient, config, httpClientHandler);
            var deviceId = "deviceId_example";  // string | The device id shown on the controlled machine.
            var metricAlertInput = new MetricAlertInput(); // MetricAlertInput | 

            try
            {
                // Create or replace the owner's metric alert rule
                MetricAlertRule result = apiInstance.PutMetricAlert(deviceId, metricAlertInput);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling NotificationsApi.PutMetricAlert: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the PutMetricAlertWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Create or replace the owner's metric alert rule
    ApiResponse<MetricAlertRule> response = apiInstance.PutMetricAlertWithHttpInfo(deviceId, metricAlertInput);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling NotificationsApi.PutMetricAlertWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **deviceId** | **string** | The device id shown on the controlled machine. |  |
| **metricAlertInput** | [**MetricAlertInput**](MetricAlertInput.md) |  |  |

### Return type

[**MetricAlertRule**](MetricAlertRule.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The stored rule and latest sample. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="putofflinealert"></a>
# **PutOfflineAlert**
> OfflineAlertRule PutOfflineAlert (string deviceId, OfflineAlertInput offlineAlertInput)

Create or replace the owner's offline-alert rule

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
    public class PutOfflineAlertExample
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
            var apiInstance = new NotificationsApi(httpClient, config, httpClientHandler);
            var deviceId = "deviceId_example";  // string | The device id shown on the controlled machine.
            var offlineAlertInput = new OfflineAlertInput(); // OfflineAlertInput | 

            try
            {
                // Create or replace the owner's offline-alert rule
                OfflineAlertRule result = apiInstance.PutOfflineAlert(deviceId, offlineAlertInput);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling NotificationsApi.PutOfflineAlert: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the PutOfflineAlertWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Create or replace the owner's offline-alert rule
    ApiResponse<OfflineAlertRule> response = apiInstance.PutOfflineAlertWithHttpInfo(deviceId, offlineAlertInput);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling NotificationsApi.PutOfflineAlertWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **deviceId** | **string** | The device id shown on the controlled machine. |  |
| **offlineAlertInput** | [**OfflineAlertInput**](OfflineAlertInput.md) |  |  |

### Return type

[**OfflineAlertRule**](OfflineAlertRule.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The stored rule. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

