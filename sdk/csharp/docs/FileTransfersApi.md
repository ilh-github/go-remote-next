# QuickDesk.Signaling.Client.Api.FileTransfersApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
|--------|--------------|-------------|
| [**CreateFileTransferAudit**](FileTransfersApi.md#createfiletransferaudit) | **POST** /v1/me/file-transfer-audits | Report a controller-observed file transfer for auditing |

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

