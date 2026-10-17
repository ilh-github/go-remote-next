# QuickDesk.Signaling.Client.Api.UpdateApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
|--------|--------------|-------------|
| [**CheckForUpdate**](UpdateApi.md#checkforupdate) | **GET** /v1/update:check | Check the self-hosted update channel for a newer build |
| [**DownloadUpdateArtifact**](UpdateApi.md#downloadupdateartifact) | **GET** /v1/update-releases/{release_id}/artifact | Download an installer this server hosts itself |

<a id="checkforupdate"></a>
# **CheckForUpdate**
> UpdateCheckResult CheckForUpdate (string platform, string arch, string? channel = null, string? currentVersion = null)

Check the self-hosted update channel for a newer build

The client-facing half of the self-hosted update channel. It is API-key gated but not user-authenticated: a controlled machine checks for updates before anyone logs in.  The server owns the index of published releases. The artifact lives at `download_url`: normally a host the operator controls, or this server itself when the installer was uploaded to it (then `download_url` is `GET /v1/update-releases/{release_id}/artifact` and `hosted_here` is true). The index is managed through the admin API (`POST /v1/admin/update-releases` and friends), which stays outside this business contract.  `current_version` is optional. Without it the newest release is returned and `update_available` is true whenever one exists; with it, `update_available` is true only when the published version is strictly newer, and `mandatory` mirrors the release flag once an update is available. `latest` is always present and is `null` when the channel has no release for that platform/arch. 

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
    public class CheckForUpdateExample
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
            var apiInstance = new UpdateApi(httpClient, config, httpClientHandler);
            var platform = "windows";  // string | Which operating system the caller is running.
            var arch = "x64";  // string | Which architecture the caller is running.
            var channel = "stable";  // string? | Defaults to `stable`. (optional) 
            var currentVersion = "currentVersion_example";  // string? | The dotted numeric version the caller is running, e.g. `4.2.5.0`. A leading `v` is accepted. (optional) 

            try
            {
                // Check the self-hosted update channel for a newer build
                UpdateCheckResult result = apiInstance.CheckForUpdate(platform, arch, channel, currentVersion);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling UpdateApi.CheckForUpdate: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the CheckForUpdateWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Check the self-hosted update channel for a newer build
    ApiResponse<UpdateCheckResult> response = apiInstance.CheckForUpdateWithHttpInfo(platform, arch, channel, currentVersion);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling UpdateApi.CheckForUpdateWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **platform** | **string** | Which operating system the caller is running. |  |
| **arch** | **string** | Which architecture the caller is running. |  |
| **channel** | **string?** | Defaults to &#x60;stable&#x60;. | [optional]  |
| **currentVersion** | **string?** | The dotted numeric version the caller is running, e.g. &#x60;4.2.5.0&#x60;. A leading &#x60;v&#x60; is accepted. | [optional]  |

### Return type

[**UpdateCheckResult**](UpdateCheckResult.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The newest release for that channel/platform/arch, or a null &#x60;latest&#x60; when the channel is empty. |  -  |
| **400** | The request was malformed or failed validation. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="downloadupdateartifact"></a>
# **DownloadUpdateArtifact**
> FileParameter DownloadUpdateArtifact (long releaseId)

Download an installer this server hosts itself

The other half of a re-hosted release. `GET /v1/update:check` returns the `download_url` of the newest build, and for a release whose bytes were uploaded to this server that URL is this route, under the same API key the client already used for the check.  API-key gated but not user-authenticated, like the check itself. The response supports range requests, so a transfer that drops can resume.  A release whose bytes live on an external static host is not served here: fetching its route is a `404`, and the client should use the `download_url` from the check response instead. 

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
    public class DownloadUpdateArtifactExample
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
            var apiInstance = new UpdateApi(httpClient, config, httpClientHandler);
            var releaseId = 789L;  // long | The `id` of the release from the check response.

            try
            {
                // Download an installer this server hosts itself
                FileParameter result = apiInstance.DownloadUpdateArtifact(releaseId);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling UpdateApi.DownloadUpdateArtifact: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the DownloadUpdateArtifactWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Download an installer this server hosts itself
    ApiResponse<FileParameter> response = apiInstance.DownloadUpdateArtifactWithHttpInfo(releaseId);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling UpdateApi.DownloadUpdateArtifactWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **releaseId** | **long** | The &#x60;id&#x60; of the release from the check response. |  |

### Return type

[**FileParameter**](FileParameter.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/octet-stream, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The installer bytes. |  * Content-Disposition - The stored file name, as an attachment. <br>  * Accept-Ranges - Present because range requests are supported. <br>  |
| **400** | The request was malformed or failed validation. |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

