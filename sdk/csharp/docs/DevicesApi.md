# QuickDesk.Signaling.Client.Api.DevicesApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
|--------|--------------|-------------|
| [**ApproveAccessApproval**](DevicesApi.md#approveaccessapproval) | **POST** /v1/me/devices/{device_id}/access-approvals/{approval_id}/approve | Approve a pending access request |
| [**BanDeviceAccessSource**](DevicesApi.md#bandeviceaccesssource) | **POST** /v1/me/devices/{device_id}/access-bans | Permanently deny one source address |
| [**ClaimAccessApproval**](DevicesApi.md#claimaccessapproval) | **POST** /v1/devices/{device_id}/access-approvals:claim | Exchange an approved request for a signal token |
| [**CreateAccessGrant**](DevicesApi.md#createaccessgrant) | **POST** /v1/me/devices/{device_id}/access-grants | Issue a time-boxed guest access code |
| [**DenyAccessApproval**](DevicesApi.md#denyaccessapproval) | **POST** /v1/me/devices/{device_id}/access-approvals/{approval_id}/deny | Deny a pending access request |
| [**GetAccessApprovalSummary**](DevicesApi.md#getaccessapprovalsummary) | **GET** /v1/me/devices/{device_id}/access-approvals:summary | Aggregated view of a device&#39;s approval queue |
| [**GetAccessPolicy**](DevicesApi.md#getaccesspolicy) | **GET** /v1/me/devices/{device_id}/access-policy | The device&#39;s access policy |
| [**ListAccessApprovals**](DevicesApi.md#listaccessapprovals) | **GET** /v1/me/devices/{device_id}/access-approvals | Second-factor approvals awaiting (or already decided by) the owner |
| [**ListAccessGrants**](DevicesApi.md#listaccessgrants) | **GET** /v1/me/devices/{device_id}/access-grants | Guest access codes issued for a device |
| [**ListMyDevices**](DevicesApi.md#listmydevices) | **GET** /v1/me/devices | Devices owned by the authenticated user |
| [**PutAccessPolicy**](DevicesApi.md#putaccesspolicy) | **PUT** /v1/me/devices/{device_id}/access-policy | Create or replace the device&#39;s access policy |
| [**ReportHostMetrics**](DevicesApi.md#reporthostmetrics) | **POST** /v1/devices/{device_id}/metrics | Report the host&#39;s latest resource sample |
| [**RevokeAccessGrant**](DevicesApi.md#revokeaccessgrant) | **DELETE** /v1/me/devices/{device_id}/access-grants/{grant_id} | Revoke a guest access code |
| [**UnbanDeviceAccessSource**](DevicesApi.md#unbandeviceaccesssource) | **POST** /v1/me/devices/{device_id}/access-bans:remove | Remove one permanent source ban |
| [**VerifyAccessCode**](DevicesApi.md#verifyaccesscode) | **POST** /v1/devices/{device_id}/access-code:verify | Verify a device access code and mint a signal token |

<a id="approveaccessapproval"></a>
# **ApproveAccessApproval**
> AccessApproval ApproveAccessApproval (string deviceId, long approvalId, ApproveAccessApprovalRequest? approveAccessApprovalRequest = null)

Approve a pending access request

Owner-only. The body is optional; a `reason` is stored on the row and shown to the requester. A request that was already decided or has expired returns `409` with the current status instead of pretending the decision landed. The conformance smoke pins the `404` shape with an unknown id; the happy path needs a real pending approval (an online device plus `approval_mode`), so it is covered by the end-to-end run in `docs/dev/自托管端到端联调记录.md` rather than the smoke. 

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
    public class ApproveAccessApprovalExample
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
            var apiInstance = new DevicesApi(httpClient, config, httpClientHandler);
            var deviceId = "deviceId_example";  // string | The device id shown on the controlled machine.
            var approvalId = 789L;  // long | The numeric id from an approval row.
            var approveAccessApprovalRequest = new ApproveAccessApprovalRequest?(); // ApproveAccessApprovalRequest? |  (optional) 

            try
            {
                // Approve a pending access request
                AccessApproval result = apiInstance.ApproveAccessApproval(deviceId, approvalId, approveAccessApprovalRequest);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling DevicesApi.ApproveAccessApproval: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the ApproveAccessApprovalWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Approve a pending access request
    ApiResponse<AccessApproval> response = apiInstance.ApproveAccessApprovalWithHttpInfo(deviceId, approvalId, approveAccessApprovalRequest);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling DevicesApi.ApproveAccessApprovalWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **deviceId** | **string** | The device id shown on the controlled machine. |  |
| **approvalId** | **long** | The numeric id from an approval row. |  |
| **approveAccessApprovalRequest** | [**ApproveAccessApprovalRequest?**](ApproveAccessApprovalRequest?.md) |  | [optional]  |

### Return type

[**AccessApproval**](AccessApproval.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The approval after the decision. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |
| **409** | The request was already decided or has expired. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="bandeviceaccesssource"></a>
# **BanDeviceAccessSource**
> DeviceAccessPolicy BanDeviceAccessSource (string deviceId, DeviceAccessBanRequest deviceAccessBanRequest)

Permanently deny one source address

Adds an exact IP or CIDR block to the device's persistent deny list and returns the complete updated policy. A deny match always wins over the allow list, time windows and a valid access code. Repeating the same source is a no-op; remove one source with `POST /v1/me/devices/{device_id}/access-bans:remove`. 

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
    public class BanDeviceAccessSourceExample
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
            var apiInstance = new DevicesApi(httpClient, config, httpClientHandler);
            var deviceId = "deviceId_example";  // string | The device id shown on the controlled machine.
            var deviceAccessBanRequest = new DeviceAccessBanRequest(); // DeviceAccessBanRequest | 

            try
            {
                // Permanently deny one source address
                DeviceAccessPolicy result = apiInstance.BanDeviceAccessSource(deviceId, deviceAccessBanRequest);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling DevicesApi.BanDeviceAccessSource: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the BanDeviceAccessSourceWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Permanently deny one source address
    ApiResponse<DeviceAccessPolicy> response = apiInstance.BanDeviceAccessSourceWithHttpInfo(deviceId, deviceAccessBanRequest);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling DevicesApi.BanDeviceAccessSourceWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **deviceId** | **string** | The device id shown on the controlled machine. |  |
| **deviceAccessBanRequest** | [**DeviceAccessBanRequest**](DeviceAccessBanRequest.md) |  |  |

### Return type

[**DeviceAccessPolicy**](DeviceAccessPolicy.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The updated policy, including the complete deny list. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="claimaccessapproval"></a>
# **ClaimAccessApproval**
> AccessCodeVerifyResponse ClaimAccessApproval (string deviceId, ApprovalClaimRequest approvalClaimRequest)

Exchange an approved request for a signal token

The requesting controller calls this with the `approval_id` and `approval_secret` it received from `access-code:verify`. It is API-key gated rather than user-authenticated because the caller has no account. `202` means the owner has not decided yet - keep polling with the same id and secret; `200` returns the signal token; `403` means denied or expired; `409` means the approval was already spent or the host went offline before the exchange. 

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
    public class ClaimAccessApprovalExample
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
            var apiInstance = new DevicesApi(httpClient, config, httpClientHandler);
            var deviceId = "deviceId_example";  // string | The device id shown on the controlled machine.
            var approvalClaimRequest = new ApprovalClaimRequest(); // ApprovalClaimRequest | 

            try
            {
                // Exchange an approved request for a signal token
                AccessCodeVerifyResponse result = apiInstance.ClaimAccessApproval(deviceId, approvalClaimRequest);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling DevicesApi.ClaimAccessApproval: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the ClaimAccessApprovalWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Exchange an approved request for a signal token
    ApiResponse<AccessCodeVerifyResponse> response = apiInstance.ClaimAccessApprovalWithHttpInfo(deviceId, approvalClaimRequest);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling DevicesApi.ClaimAccessApprovalWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **deviceId** | **string** | The device id shown on the controlled machine. |  |
| **approvalClaimRequest** | [**ApprovalClaimRequest**](ApprovalClaimRequest.md) |  |  |

### Return type

[**AccessCodeVerifyResponse**](AccessCodeVerifyResponse.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The owner approved and the signal token is ready. |  -  |
| **202** | Still waiting for the owner to decide. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |
| **409** | The approval was already spent, or the host went offline before the exchange could happen.  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="createaccessgrant"></a>
# **CreateAccessGrant**
> AccessGrant CreateAccessGrant (string deviceId, AccessGrantInput accessGrantInput)

Issue a time-boxed guest access code

The plaintext code is returned **once**, in the creation response; a grant is limited by expiry and uses, may be bound to a client id and/or source addresses, and is revoked with the DELETE below. 

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
    public class CreateAccessGrantExample
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
            var apiInstance = new DevicesApi(httpClient, config, httpClientHandler);
            var deviceId = "deviceId_example";  // string | The device id shown on the controlled machine.
            var accessGrantInput = new AccessGrantInput(); // AccessGrantInput | 

            try
            {
                // Issue a time-boxed guest access code
                AccessGrant result = apiInstance.CreateAccessGrant(deviceId, accessGrantInput);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling DevicesApi.CreateAccessGrant: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the CreateAccessGrantWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Issue a time-boxed guest access code
    ApiResponse<AccessGrant> response = apiInstance.CreateAccessGrantWithHttpInfo(deviceId, accessGrantInput);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling DevicesApi.CreateAccessGrantWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **deviceId** | **string** | The device id shown on the controlled machine. |  |
| **accessGrantInput** | [**AccessGrantInput**](AccessGrantInput.md) |  |  |

### Return type

[**AccessGrant**](AccessGrant.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **201** | The created grant, including its one-time plaintext code. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="denyaccessapproval"></a>
# **DenyAccessApproval**
> AccessApproval DenyAccessApproval (string deviceId, long approvalId, ApproveAccessApprovalRequest? approveAccessApprovalRequest = null)

Deny a pending access request

Owner-only, same shape as `approve`. Denying also arms the per-source denial cooldown (`APPROVAL_DENIAL_COOLDOWN_SECONDS`), so the same client cannot immediately re-request. 

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
    public class DenyAccessApprovalExample
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
            var apiInstance = new DevicesApi(httpClient, config, httpClientHandler);
            var deviceId = "deviceId_example";  // string | The device id shown on the controlled machine.
            var approvalId = 789L;  // long | The numeric id from an approval row.
            var approveAccessApprovalRequest = new ApproveAccessApprovalRequest?(); // ApproveAccessApprovalRequest? |  (optional) 

            try
            {
                // Deny a pending access request
                AccessApproval result = apiInstance.DenyAccessApproval(deviceId, approvalId, approveAccessApprovalRequest);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling DevicesApi.DenyAccessApproval: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the DenyAccessApprovalWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Deny a pending access request
    ApiResponse<AccessApproval> response = apiInstance.DenyAccessApprovalWithHttpInfo(deviceId, approvalId, approveAccessApprovalRequest);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling DevicesApi.DenyAccessApprovalWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **deviceId** | **string** | The device id shown on the controlled machine. |  |
| **approvalId** | **long** | The numeric id from an approval row. |  |
| **approveAccessApprovalRequest** | [**ApproveAccessApprovalRequest?**](ApproveAccessApprovalRequest?.md) |  | [optional]  |

### Return type

[**AccessApproval**](AccessApproval.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The approval after the decision. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |
| **409** | The request was already decided or has expired. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="getaccessapprovalsummary"></a>
# **GetAccessApprovalSummary**
> ApprovalSummary GetAccessApprovalSummary (string deviceId, DateTime? since = null, int? limit = null)

Aggregated view of a device's approval queue

The row list shows requests one at a time; this is what makes \"one address has been refused six times this afternoon\" legible. The window is keyed on `created_at` for every count. 

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
    public class GetAccessApprovalSummaryExample
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
            var apiInstance = new DevicesApi(httpClient, config, httpClientHandler);
            var deviceId = "deviceId_example";  // string | The device id shown on the controlled machine.
            var since = DateTime.Parse("2013-10-20T19:20:30+01:00");  // DateTime? | RFC3339 start of the window; defaults to the last 24 hours. (optional) 
            var limit = 56;  // int? | How many refused sources to list (default 5, capped at 10). (optional) 

            try
            {
                // Aggregated view of a device's approval queue
                ApprovalSummary result = apiInstance.GetAccessApprovalSummary(deviceId, since, limit);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling DevicesApi.GetAccessApprovalSummary: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the GetAccessApprovalSummaryWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Aggregated view of a device's approval queue
    ApiResponse<ApprovalSummary> response = apiInstance.GetAccessApprovalSummaryWithHttpInfo(deviceId, since, limit);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling DevicesApi.GetAccessApprovalSummaryWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **deviceId** | **string** | The device id shown on the controlled machine. |  |
| **since** | **DateTime?** | RFC3339 start of the window; defaults to the last 24 hours. | [optional]  |
| **limit** | **int?** | How many refused sources to list (default 5, capped at 10). | [optional]  |

### Return type

[**ApprovalSummary**](ApprovalSummary.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The window&#39;s counts and refused sources. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="getaccesspolicy"></a>
# **GetAccessPolicy**
> DeviceAccessPolicy GetAccessPolicy (string deviceId)

The device's access policy

Returns the documented \"no policy\" defaults when the owner never saved one, so a client can render the form without a second existence check. 

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
    public class GetAccessPolicyExample
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
            var apiInstance = new DevicesApi(httpClient, config, httpClientHandler);
            var deviceId = "deviceId_example";  // string | The device id shown on the controlled machine.

            try
            {
                // The device's access policy
                DeviceAccessPolicy result = apiInstance.GetAccessPolicy(deviceId);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling DevicesApi.GetAccessPolicy: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the GetAccessPolicyWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // The device's access policy
    ApiResponse<DeviceAccessPolicy> response = apiInstance.GetAccessPolicyWithHttpInfo(deviceId);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling DevicesApi.GetAccessPolicyWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **deviceId** | **string** | The device id shown on the controlled machine. |  |

### Return type

[**DeviceAccessPolicy**](DeviceAccessPolicy.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The policy, or its defaults. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="listaccessapprovals"></a>
# **ListAccessApprovals**
> ListAccessApprovals200Response ListAccessApprovals (string deviceId, string? status = null, int? limit = null)

Second-factor approvals awaiting (or already decided by) the owner

`status` is a comma-separated subset of `pending, approved, denied, expired, consumed` and defaults to `pending`. Approval is created by the device-side verify route when the policy asks for it; approving or denying happens on the sub-resource routes documented in `docs/user-api-docs.md` §5.3.5 (not yet part of this contract). 

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
    public class ListAccessApprovalsExample
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
            var apiInstance = new DevicesApi(httpClient, config, httpClientHandler);
            var deviceId = "deviceId_example";  // string | The device id shown on the controlled machine.
            var status = "status_example";  // string? |  (optional) 
            var limit = 56;  // int? |  (optional) 

            try
            {
                // Second-factor approvals awaiting (or already decided by) the owner
                ListAccessApprovals200Response result = apiInstance.ListAccessApprovals(deviceId, status, limit);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling DevicesApi.ListAccessApprovals: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the ListAccessApprovalsWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Second-factor approvals awaiting (or already decided by) the owner
    ApiResponse<ListAccessApprovals200Response> response = apiInstance.ListAccessApprovalsWithHttpInfo(deviceId, status, limit);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling DevicesApi.ListAccessApprovalsWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **deviceId** | **string** | The device id shown on the controlled machine. |  |
| **status** | **string?** |  | [optional]  |
| **limit** | **int?** |  | [optional]  |

### Return type

[**ListAccessApprovals200Response**](ListAccessApprovals200Response.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The matching approvals, newest first. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="listaccessgrants"></a>
# **ListAccessGrants**
> ListAccessGrants200Response ListAccessGrants (string deviceId)

Guest access codes issued for a device

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
    public class ListAccessGrantsExample
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
            var apiInstance = new DevicesApi(httpClient, config, httpClientHandler);
            var deviceId = "deviceId_example";  // string | The device id shown on the controlled machine.

            try
            {
                // Guest access codes issued for a device
                ListAccessGrants200Response result = apiInstance.ListAccessGrants(deviceId);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling DevicesApi.ListAccessGrants: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the ListAccessGrantsWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Guest access codes issued for a device
    ApiResponse<ListAccessGrants200Response> response = apiInstance.ListAccessGrantsWithHttpInfo(deviceId);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling DevicesApi.ListAccessGrantsWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **deviceId** | **string** | The device id shown on the controlled machine. |  |

### Return type

[**ListAccessGrants200Response**](ListAccessGrants200Response.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The device&#39;s grants, newest first. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="listmydevices"></a>
# **ListMyDevices**
> ListMyDevices200Response ListMyDevices ()

Devices owned by the authenticated user

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
    public class ListMyDevicesExample
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
            var apiInstance = new DevicesApi(httpClient, config, httpClientHandler);

            try
            {
                // Devices owned by the authenticated user
                ListMyDevices200Response result = apiInstance.ListMyDevices();
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling DevicesApi.ListMyDevices: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the ListMyDevicesWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Devices owned by the authenticated user
    ApiResponse<ListMyDevices200Response> response = apiInstance.ListMyDevicesWithHttpInfo();
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling DevicesApi.ListMyDevicesWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters
This endpoint does not need any parameter.
### Return type

[**ListMyDevices200Response**](ListMyDevices200Response.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The caller&#39;s devices. |  -  |
| **401** | Missing or expired credentials. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="putaccesspolicy"></a>
# **PutAccessPolicy**
> DeviceAccessPolicy PutAccessPolicy (string deviceId, DeviceAccessPolicyInput deviceAccessPolicyInput)

Create or replace the device's access policy

A **full replace**: every field not sent is reset to its zero value. The policy is enforced on the device-side verify route before the access code is compared, so a blocked client never gets to guess codes. 

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
    public class PutAccessPolicyExample
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
            var apiInstance = new DevicesApi(httpClient, config, httpClientHandler);
            var deviceId = "deviceId_example";  // string | The device id shown on the controlled machine.
            var deviceAccessPolicyInput = new DeviceAccessPolicyInput(); // DeviceAccessPolicyInput | 

            try
            {
                // Create or replace the device's access policy
                DeviceAccessPolicy result = apiInstance.PutAccessPolicy(deviceId, deviceAccessPolicyInput);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling DevicesApi.PutAccessPolicy: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the PutAccessPolicyWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Create or replace the device's access policy
    ApiResponse<DeviceAccessPolicy> response = apiInstance.PutAccessPolicyWithHttpInfo(deviceId, deviceAccessPolicyInput);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling DevicesApi.PutAccessPolicyWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **deviceId** | **string** | The device id shown on the controlled machine. |  |
| **deviceAccessPolicyInput** | [**DeviceAccessPolicyInput**](DeviceAccessPolicyInput.md) |  |  |

### Return type

[**DeviceAccessPolicy**](DeviceAccessPolicy.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The stored policy. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="reporthostmetrics"></a>
# **ReportHostMetrics**
> ReportHostMetrics200Response ReportHostMetrics (string deviceId, HostMetricReport hostMetricReport)

Report the host's latest resource sample

Host-only endpoint authenticated by both the deployment API key and the device secret. Values are percentages in the inclusive range 0..100. Repeating a report replaces the previous sample and advances the owner's metric-alert episode state. 

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
    public class ReportHostMetricsExample
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
            var apiInstance = new DevicesApi(httpClient, config, httpClientHandler);
            var deviceId = "deviceId_example";  // string | The device id shown on the controlled machine.
            var hostMetricReport = new HostMetricReport(); // HostMetricReport | 

            try
            {
                // Report the host's latest resource sample
                ReportHostMetrics200Response result = apiInstance.ReportHostMetrics(deviceId, hostMetricReport);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling DevicesApi.ReportHostMetrics: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the ReportHostMetricsWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Report the host's latest resource sample
    ApiResponse<ReportHostMetrics200Response> response = apiInstance.ReportHostMetricsWithHttpInfo(deviceId, hostMetricReport);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling DevicesApi.ReportHostMetricsWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **deviceId** | **string** | The device id shown on the controlled machine. |  |
| **hostMetricReport** | [**HostMetricReport**](HostMetricReport.md) |  |  |

### Return type

[**ReportHostMetrics200Response**](ReportHostMetrics200Response.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The accepted sample identity. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="revokeaccessgrant"></a>
# **RevokeAccessGrant**
> RevokeAccessGrant200Response RevokeAccessGrant (string deviceId, long grantId)

Revoke a guest access code

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
    public class RevokeAccessGrantExample
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
            var apiInstance = new DevicesApi(httpClient, config, httpClientHandler);
            var deviceId = "deviceId_example";  // string | The device id shown on the controlled machine.
            var grantId = 789L;  // long | 

            try
            {
                // Revoke a guest access code
                RevokeAccessGrant200Response result = apiInstance.RevokeAccessGrant(deviceId, grantId);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling DevicesApi.RevokeAccessGrant: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the RevokeAccessGrantWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Revoke a guest access code
    ApiResponse<RevokeAccessGrant200Response> response = apiInstance.RevokeAccessGrantWithHttpInfo(deviceId, grantId);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling DevicesApi.RevokeAccessGrantWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **deviceId** | **string** | The device id shown on the controlled machine. |  |
| **grantId** | **long** |  |  |

### Return type

[**RevokeAccessGrant200Response**](RevokeAccessGrant200Response.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The grant is no longer redeemable. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="unbandeviceaccesssource"></a>
# **UnbanDeviceAccessSource**
> DeviceAccessPolicy UnbanDeviceAccessSource (string deviceId, DeviceAccessBanRequest deviceAccessBanRequest)

Remove one permanent source ban

Removes the exact IP or CIDR entry from the device's persistent deny list and returns the complete updated policy. A source that is already absent, including on a device with no stored policy, is a successful no-op. 

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
    public class UnbanDeviceAccessSourceExample
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
            var apiInstance = new DevicesApi(httpClient, config, httpClientHandler);
            var deviceId = "deviceId_example";  // string | The device id shown on the controlled machine.
            var deviceAccessBanRequest = new DeviceAccessBanRequest(); // DeviceAccessBanRequest | 

            try
            {
                // Remove one permanent source ban
                DeviceAccessPolicy result = apiInstance.UnbanDeviceAccessSource(deviceId, deviceAccessBanRequest);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling DevicesApi.UnbanDeviceAccessSource: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the UnbanDeviceAccessSourceWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Remove one permanent source ban
    ApiResponse<DeviceAccessPolicy> response = apiInstance.UnbanDeviceAccessSourceWithHttpInfo(deviceId, deviceAccessBanRequest);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling DevicesApi.UnbanDeviceAccessSourceWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **deviceId** | **string** | The device id shown on the controlled machine. |  |
| **deviceAccessBanRequest** | [**DeviceAccessBanRequest**](DeviceAccessBanRequest.md) |  |  |

### Return type

[**DeviceAccessPolicy**](DeviceAccessPolicy.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The updated policy after the source is absent. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="verifyaccesscode"></a>
# **VerifyAccessCode**
> AccessCodeVerifyResponse VerifyAccessCode (string deviceId, AccessCodeVerifyRequest accessCodeVerifyRequest)

Verify a device access code and mint a signal token

The controller-side entry point. On success it returns a signal token for the relay. When the device policy wants the owner to confirm the session it returns `202` with a pending approval instead, and the caller polls `/v1/devices/{device_id}/access-approvals:claim`. A rejected code is a `403` problem with `code: INVALID_CODE`.  This route is reachable without a user token (the controlling client is not logged in) but still requires the API key or an allowed Origin. 

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
    public class VerifyAccessCodeExample
    {
        public static void Main()
        {
            Configuration config = new Configuration();
            config.BasePath = "https://your-signaling-host";
            // Configure API key authorization: ApiKeyAuth
            config.AddApiKey("X-API-Key", "YOUR_API_KEY");
            // Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
            // config.AddApiKeyPrefix("X-API-Key", "Bearer");

            // create instances of HttpClient, HttpClientHandler to be reused later with different Api classes
            HttpClient httpClient = new HttpClient();
            HttpClientHandler httpClientHandler = new HttpClientHandler();
            var apiInstance = new DevicesApi(httpClient, config, httpClientHandler);
            var deviceId = "deviceId_example";  // string | The device id shown on the controlled machine.
            var accessCodeVerifyRequest = new AccessCodeVerifyRequest(); // AccessCodeVerifyRequest | 

            try
            {
                // Verify a device access code and mint a signal token
                AccessCodeVerifyResponse result = apiInstance.VerifyAccessCode(deviceId, accessCodeVerifyRequest);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling DevicesApi.VerifyAccessCode: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the VerifyAccessCodeWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Verify a device access code and mint a signal token
    ApiResponse<AccessCodeVerifyResponse> response = apiInstance.VerifyAccessCodeWithHttpInfo(deviceId, accessCodeVerifyRequest);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling DevicesApi.VerifyAccessCodeWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **deviceId** | **string** | The device id shown on the controlled machine. |  |
| **accessCodeVerifyRequest** | [**AccessCodeVerifyRequest**](AccessCodeVerifyRequest.md) |  |  |

### Return type

[**AccessCodeVerifyResponse**](AccessCodeVerifyResponse.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The code was accepted and a signal token was minted. |  -  |
| **202** | The device policy requires the owner to approve this session. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |
| **429** | Too many requests; retry after the advertised delay. |  * Retry-After - Seconds to wait before retrying. <br>  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

