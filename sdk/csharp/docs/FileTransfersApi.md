# QuickDesk.Signaling.Client.Api.FileTransfersApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
|--------|--------------|-------------|
| [**CreateClipboardAudit**](FileTransfersApi.md#createclipboardaudit) | **POST** /v1/me/clipboard-audits | Report a controller-observed clipboard sync for auditing |
| [**CreateFileTransferAudit**](FileTransfersApi.md#createfiletransferaudit) | **POST** /v1/me/file-transfer-audits | Report a controller-observed file transfer for auditing |
| [**ListMyAuditLogs**](FileTransfersApi.md#listmyauditlogs) | **GET** /v1/me/audit-logs | Read back the data-channel audit trail that belongs to the caller |

<a id="createclipboardaudit"></a>
# **CreateClipboardAudit**
> void CreateClipboardAudit (ClipboardAuditInput clipboardAuditInput)

Report a controller-observed clipboard sync for auditing

Clipboard bytes travel directly over the Remoting data channel and never pass through the signaling server. This endpoint lets an authenticated device owner report a clipboard sync after the fact so it appears in the audit trail.  The report carries **metadata only**: direction, MIME type and byte count. There is deliberately no field for the clipboard text. A clipboard is by definition whatever the user last copied - routinely a password, a recovery code or a private message - so a row that stored the text would turn the audit trail into a credential store every later reader of the database inherits. \"The operator copied 1.2 KB of text at 14:03\" answers the investigative question; the text answers nothing an investigator needs and costs everything if the database leaks.  Reports are idempotent by `transfer_id`, and the device must belong to the caller. This is an attributable controller report, not cryptographic proof from the host that a clipboard sync happened. 

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
    public class CreateClipboardAuditExample
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
            var apiInstance = new FileTransfersApi(httpClient, config, httpClientHandler);
            var clipboardAuditInput = new ClipboardAuditInput(); // ClipboardAuditInput | 

            try
            {
                // Report a controller-observed clipboard sync for auditing
                apiInstance.CreateClipboardAudit(clipboardAuditInput);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling FileTransfersApi.CreateClipboardAudit: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the CreateClipboardAuditWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Report a controller-observed clipboard sync for auditing
    apiInstance.CreateClipboardAuditWithHttpInfo(clipboardAuditInput);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling FileTransfersApi.CreateClipboardAuditWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **clipboardAuditInput** | [**ClipboardAuditInput**](ClipboardAuditInput.md) |  |  |

### Return type

void (empty response body)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **204** | The clipboard report is in the audit trail (or was already recorded). |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="createfiletransferaudit"></a>
# **CreateFileTransferAudit**
> void CreateFileTransferAudit (FileTransferAuditInput fileTransferAuditInput)

Report a controller-observed file transfer for auditing

File bytes travel directly over the Remoting data channel and never pass through the signaling server. This endpoint lets an authenticated device owner report a completed or failed transfer after the fact so it appears in the audit trail.  Reports are idempotent by `(actor, direction, transfer_id)`: retrying the same event does not create another row. The device must belong to the caller. This is an attributable controller report, not cryptographic proof from the host that a file operation happened. 

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
    public class CreateFileTransferAuditExample
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
            var apiInstance = new FileTransfersApi(httpClient, config, httpClientHandler);
            var fileTransferAuditInput = new FileTransferAuditInput(); // FileTransferAuditInput | 

            try
            {
                // Report a controller-observed file transfer for auditing
                apiInstance.CreateFileTransferAudit(fileTransferAuditInput);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling FileTransfersApi.CreateFileTransferAudit: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the CreateFileTransferAuditWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Report a controller-observed file transfer for auditing
    apiInstance.CreateFileTransferAuditWithHttpInfo(fileTransferAuditInput);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling FileTransfersApi.CreateFileTransferAuditWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **fileTransferAuditInput** | [**FileTransferAuditInput**](FileTransferAuditInput.md) |  |  |

### Return type

void (empty response body)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **204** | The transfer report is in the audit trail (or was already recorded). |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="listmyauditlogs"></a>
# **ListMyAuditLogs**
> OwnerAuditLogPage ListMyAuditLogs (string? action = null, string? cursor = null, int? limit = null)

Read back the data-channel audit trail that belongs to the caller

The read side of the two reporting endpoints above. File and clipboard transfers travel peer to peer over the Remoting data channel, so the control plane can only record what a controller tells it afterwards; this is how the device owner reads those records back without going through the administrator console.  Rows are scoped to the caller's own actor id. Both reporting endpoints refuse a device the caller does not own, so the actor on a row is that device's owner and this scope cannot expose anybody else's transfers.  `action` is an allow-list rather than a free-form filter: an action this trail does not hold is refused with 400 rather than answered with an empty page, because \"you did nothing\" must not be a conclusion a typo can produce. Supported values are `file.transfer.upload`, `file.transfer.download`, `clipboard.sync.upload` and `clipboard.sync.download`.  Cursor-paginated, newest first. `next_cursor` is opaque: pass it back as `?cursor=` to fetch the next page and stop when it is absent. 

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
    public class ListMyAuditLogsExample
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
            var apiInstance = new FileTransfersApi(httpClient, config, httpClientHandler);
            var action = "file.transfer.upload";  // string? | Narrow to one audit action. An unsupported value is a 400. (optional) 
            var cursor = "cursor_example";  // string? |  (optional) 
            var limit = 56;  // int? |  (optional) 

            try
            {
                // Read back the data-channel audit trail that belongs to the caller
                OwnerAuditLogPage result = apiInstance.ListMyAuditLogs(action, cursor, limit);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling FileTransfersApi.ListMyAuditLogs: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the ListMyAuditLogsWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Read back the data-channel audit trail that belongs to the caller
    ApiResponse<OwnerAuditLogPage> response = apiInstance.ListMyAuditLogsWithHttpInfo(action, cursor, limit);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling FileTransfersApi.ListMyAuditLogsWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **action** | **string?** | Narrow to one audit action. An unsupported value is a 400. | [optional]  |
| **cursor** | **string?** |  | [optional]  |
| **limit** | **int?** |  | [optional]  |

### Return type

[**OwnerAuditLogPage**](OwnerAuditLogPage.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | One page of the caller&#39;s audit rows, newest first. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

