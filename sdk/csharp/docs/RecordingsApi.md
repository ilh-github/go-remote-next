# QuickDesk.Signaling.Client.Api.RecordingsApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
|--------|--------------|-------------|
| [**AppendRecordingArtifact**](RecordingsApi.md#appendrecordingartifact) | **PATCH** /v1/me/recordings/{recording_id}/artifact | Append one chunk of a resumable recording upload |
| [**CreateRecordingArtifactTicket**](RecordingsApi.md#createrecordingartifactticket) | **POST** /v1/me/recordings/{recording_id}/artifact-ticket | Mint a short-lived playback URL for a recording |
| [**CreateRecordingThumbnailTicket**](RecordingsApi.md#createrecordingthumbnailticket) | **POST** /v1/me/recordings/{recording_id}/thumbnail-ticket | Mint a short-lived preview URL for a recording |
| [**CreateSessionRecording**](RecordingsApi.md#createsessionrecording) | **POST** /v1/me/recordings | Index a new recording |
| [**DeleteSessionRecording**](RecordingsApi.md#deletesessionrecording) | **DELETE** /v1/me/recordings/{recording_id} | Delete a recording and its stored media |
| [**DownloadRecordingArtifact**](RecordingsApi.md#downloadrecordingartifact) | **GET** /v1/me/recordings/{recording_id}/artifact | Download a recording&#39;s media bytes |
| [**DownloadRecordingArtifactByTicket**](RecordingsApi.md#downloadrecordingartifactbyticket) | **GET** /v1/recordings/{recording_id}/artifact | Fetch recording media with a playback ticket |
| [**DownloadRecordingThumbnail**](RecordingsApi.md#downloadrecordingthumbnail) | **GET** /v1/me/recordings/{recording_id}/thumbnail | Download a recording&#39;s preview image |
| [**DownloadRecordingThumbnailByTicket**](RecordingsApi.md#downloadrecordingthumbnailbyticket) | **GET** /v1/recordings/{recording_id}/thumbnail | Fetch a recording preview with a playback ticket |
| [**GetSessionRecording**](RecordingsApi.md#getsessionrecording) | **GET** /v1/me/recordings/{recording_id} | One recording the caller owns |
| [**ListSessionRecordings**](RecordingsApi.md#listsessionrecordings) | **GET** /v1/me/recordings | The caller&#39;s session-recording index |
| [**UpdateSessionRecording**](RecordingsApi.md#updatesessionrecording) | **PATCH** /v1/me/recordings/{recording_id} | Update a recording&#39;s lifecycle metadata |
| [**UploadRecordingArtifact**](RecordingsApi.md#uploadrecordingartifact) | **POST** /v1/me/recordings/{recording_id}/artifact | Upload a recording&#39;s media bytes |
| [**UploadRecordingThumbnail**](RecordingsApi.md#uploadrecordingthumbnail) | **POST** /v1/me/recordings/{recording_id}/thumbnail | Upload a recording&#39;s preview image |

<a id="appendrecordingartifact"></a>
# **AppendRecordingArtifact**
> SessionRecording AppendRecordingArtifact (string ext, long uploadOffset, long recordingId, string? uploadLength = null, string? uploadComplete = null, string? uploadProbe = null, FileParameter? body = null)

Append one chunk of a resumable recording upload

The resumable half of the upload contract, for a capture that does not fit in one request body or a connection that drops halfway. The client sends chunks; the server answers with the offset it actually holds in the `Upload-Offset` response header, so a retry resumes from that number instead of starting over.  Three headers drive it. `Upload-Offset` (required) is where the chunk starts and must equal the bytes already staged; otherwise the answer is `409` with the authoritative offset in the same header and the client resyncs. `Upload-Length` (optional, `*` while unknown) declares the total size, which lets the server finish by itself when the last byte lands and lets it refuse a chunk that would overshoot. `Upload-Complete: 1` says this chunk ends the upload, which is how a client finishes without ever declaring a total.  A status probe asks where the upload stands without staging bytes. Send `Upload-Probe: 1` with a valid `Upload-Offset` and no body; the answer is `202` with the current offset, so a client that crashed can re-learn where it stopped without a second endpoint. An empty body with a valid offset does the same thing for raw-HTTP callers, but a generated client cannot send a zero-length body, which is why the explicit header exists.  The staging file (`<artifact-path>.part`) is subject to the same size limit as the single-shot upload, an empty artifact is refused on every path, and the recording lifecycle deletes a staging file that has been untouched for six hours. 

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
    public class AppendRecordingArtifactExample
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
            var apiInstance = new RecordingsApi(httpClient, config, httpClientHandler);
            var ext = ".mp4";  // string | The artifact extension, allow-listed exactly like the raw-body POST shape.
            var uploadOffset = 789L;  // long | The byte offset this chunk starts at.
            var recordingId = 789L;  // long | The numeric id from a recording index row.
            var uploadLength = "uploadLength_example";  // string? | The total size of the artifact, or `*` while it is unknown. (optional) 
            var uploadComplete = "0";  // string? | Set to `1` when this chunk ends the upload. (optional) 
            var uploadProbe = "0";  // string? | Set to `1` to ask for the current offset without staging any bytes. The generated clients use this instead of an empty body. (optional) 
            var body = new System.IO.MemoryStream(System.IO.File.ReadAllBytes("/path/to/file.txt"));  // FileParameter? |  (optional) 

            try
            {
                // Append one chunk of a resumable recording upload
                SessionRecording result = apiInstance.AppendRecordingArtifact(ext, uploadOffset, recordingId, uploadLength, uploadComplete, uploadProbe, body);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling RecordingsApi.AppendRecordingArtifact: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the AppendRecordingArtifactWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Append one chunk of a resumable recording upload
    ApiResponse<SessionRecording> response = apiInstance.AppendRecordingArtifactWithHttpInfo(ext, uploadOffset, recordingId, uploadLength, uploadComplete, uploadProbe, body);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling RecordingsApi.AppendRecordingArtifactWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **ext** | **string** | The artifact extension, allow-listed exactly like the raw-body POST shape. |  |
| **uploadOffset** | **long** | The byte offset this chunk starts at. |  |
| **recordingId** | **long** | The numeric id from a recording index row. |  |
| **uploadLength** | **string?** | The total size of the artifact, or &#x60;*&#x60; while it is unknown. | [optional]  |
| **uploadComplete** | **string?** | Set to &#x60;1&#x60; when this chunk ends the upload. | [optional]  |
| **uploadProbe** | **string?** | Set to &#x60;1&#x60; to ask for the current offset without staging any bytes. The generated clients use this instead of an empty body. | [optional]  |
| **body** | **FileParameter?****FileParameter?** |  | [optional]  |

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/octet-stream
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The final chunk; the recording now carries the stored artifact. |  * Upload-Offset - Bytes staged, which for a finished upload is the artifact size. <br>  |
| **202** | The chunk was staged and more are expected. The authoritative offset is the &#x60;Upload-Offset&#x60; response header. The body is the recording row - the same &#x60;SessionRecording&#x60; schema as the &#x60;200&#x60; - because a generated client declares one success type per operation and would otherwise fail to decode a staging-only body.  |  * Upload-Offset - Bytes staged, which for a finished upload is the artifact size. <br>  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |
| **409** | &#x60;Upload-Offset&#x60; did not match the bytes already staged. The header carries the authoritative offset, so the client can resume from it.  |  * Upload-Offset - Bytes staged, which for a finished upload is the artifact size. <br>  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="createrecordingartifactticket"></a>
# **CreateRecordingArtifactTicket**
> PlaybackTicket CreateRecordingArtifactTicket (long recordingId)

Mint a short-lived playback URL for a recording

Owner-only. Returns a signed, time-limited `url` that can be fetched **without** an `Authorization` header, which is what makes playback work from `<video src>`. The ticket is bound to the recording, the owner and its purpose, so an artifact ticket cannot be replayed against the thumbnail route. Minting is stateless - nothing is written - and the link stops working when `expires_at` passes. `400` when playback tickets are disabled on the deployment (`RECORDING_URL_SECRET`/`API_KEY` unset); `404` when the recording has no stored artifact. 

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
    public class CreateRecordingArtifactTicketExample
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
            var apiInstance = new RecordingsApi(httpClient, config, httpClientHandler);
            var recordingId = 789L;  // long | The numeric id from a recording index row.

            try
            {
                // Mint a short-lived playback URL for a recording
                PlaybackTicket result = apiInstance.CreateRecordingArtifactTicket(recordingId);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling RecordingsApi.CreateRecordingArtifactTicket: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the CreateRecordingArtifactTicketWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Mint a short-lived playback URL for a recording
    ApiResponse<PlaybackTicket> response = apiInstance.CreateRecordingArtifactTicketWithHttpInfo(recordingId);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling RecordingsApi.CreateRecordingArtifactTicketWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **recordingId** | **long** | The numeric id from a recording index row. |  |

### Return type

[**PlaybackTicket**](PlaybackTicket.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | A signed playback URL. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="createrecordingthumbnailticket"></a>
# **CreateRecordingThumbnailTicket**
> PlaybackTicket CreateRecordingThumbnailTicket (long recordingId)

Mint a short-lived preview URL for a recording

Same rules as the artifact ticket, but the URL addresses the thumbnail route and the ticket's purpose is checked on redemption. 

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
    public class CreateRecordingThumbnailTicketExample
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
            var apiInstance = new RecordingsApi(httpClient, config, httpClientHandler);
            var recordingId = 789L;  // long | The numeric id from a recording index row.

            try
            {
                // Mint a short-lived preview URL for a recording
                PlaybackTicket result = apiInstance.CreateRecordingThumbnailTicket(recordingId);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling RecordingsApi.CreateRecordingThumbnailTicket: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the CreateRecordingThumbnailTicketWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Mint a short-lived preview URL for a recording
    ApiResponse<PlaybackTicket> response = apiInstance.CreateRecordingThumbnailTicketWithHttpInfo(recordingId);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling RecordingsApi.CreateRecordingThumbnailTicketWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **recordingId** | **long** | The numeric id from a recording index row. |  |

### Return type

[**PlaybackTicket**](PlaybackTicket.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | A signed preview URL. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="createsessionrecording"></a>
# **CreateSessionRecording**
> SessionRecording CreateSessionRecording (SessionRecordingInput sessionRecordingInput)

Index a new recording

Called by the capturing side when a session starts (or when a finished capture is handed over), so the owner sees the recording while it is still being written and the row can carry the outcome afterwards. `status` is one of `recording`, `completed` or `failed`; `started_at` defaults to now. The media is uploaded separately to `.../artifact`.  The row belongs to the *caller*: `device_id` is recorded as a label and is not checked for ownership, so a producer can index a capture for a machine that is not bound to it. Only the caller can then read or delete the row. 

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
    public class CreateSessionRecordingExample
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
            var apiInstance = new RecordingsApi(httpClient, config, httpClientHandler);
            var sessionRecordingInput = new SessionRecordingInput(); // SessionRecordingInput | 

            try
            {
                // Index a new recording
                SessionRecording result = apiInstance.CreateSessionRecording(sessionRecordingInput);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling RecordingsApi.CreateSessionRecording: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the CreateSessionRecordingWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Index a new recording
    ApiResponse<SessionRecording> response = apiInstance.CreateSessionRecordingWithHttpInfo(sessionRecordingInput);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling RecordingsApi.CreateSessionRecordingWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **sessionRecordingInput** | [**SessionRecordingInput**](SessionRecordingInput.md) |  |  |

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **201** | The created index row. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="deletesessionrecording"></a>
# **DeleteSessionRecording**
> void DeleteSessionRecording (long recordingId)

Delete a recording and its stored media

Removes the index row and the media files that belong to it, so the recording cannot be played back afterwards. There is no \"already gone\" success: a second delete is a `404`. The deletion is published on the owner's realtime stream and written to the audit trail as `recording.delete` with the actor's observed address. 

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
    public class DeleteSessionRecordingExample
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
            var apiInstance = new RecordingsApi(httpClient, config, httpClientHandler);
            var recordingId = 789L;  // long | The numeric id from a recording index row.

            try
            {
                // Delete a recording and its stored media
                apiInstance.DeleteSessionRecording(recordingId);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling RecordingsApi.DeleteSessionRecording: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the DeleteSessionRecordingWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Delete a recording and its stored media
    apiInstance.DeleteSessionRecordingWithHttpInfo(recordingId);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling RecordingsApi.DeleteSessionRecordingWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **recordingId** | **long** | The numeric id from a recording index row. |  |

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
| **204** | Deleted. No body. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="downloadrecordingartifact"></a>
# **DownloadRecordingArtifact**
> FileParameter DownloadRecordingArtifact (long recordingId)

Download a recording's media bytes

Owner-only, `Authorization` (Bearer) required. The response is the stored file itself: the `Content-Type` follows the stored file's extension (`video/mp4` for `.mp4`, `application/octet-stream` otherwise) and `Content-Disposition` is `inline`. A browser cannot attach an `Authorization` header to `<video src>`, which is why the ticket routes below exist; this route is for API clients. 

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
    public class DownloadRecordingArtifactExample
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
            var apiInstance = new RecordingsApi(httpClient, config, httpClientHandler);
            var recordingId = 789L;  // long | The numeric id from a recording index row.

            try
            {
                // Download a recording's media bytes
                FileParameter result = apiInstance.DownloadRecordingArtifact(recordingId);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling RecordingsApi.DownloadRecordingArtifact: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the DownloadRecordingArtifactWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Download a recording's media bytes
    ApiResponse<FileParameter> response = apiInstance.DownloadRecordingArtifactWithHttpInfo(recordingId);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling RecordingsApi.DownloadRecordingArtifactWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **recordingId** | **long** | The numeric id from a recording index row. |  |

### Return type

[**FileParameter**](FileParameter.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: video/mp4, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The media bytes. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="downloadrecordingartifactbyticket"></a>
# **DownloadRecordingArtifactByTicket**
> FileParameter DownloadRecordingArtifactByTicket (string t, long recordingId)

Fetch recording media with a playback ticket

The URL returned by `POST /v1/me/recordings/{recording_id}/artifact-ticket`. Deliberately registered **outside** the API-key and user-auth groups, because a browser `<video>` element can send neither header - the signed `t` query parameter is the authorization, and the ticket is bound to owner and purpose. `401` for a missing, malformed or expired ticket; `404` when the ticket is for the wrong route or the recording is gone. 

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
    public class DownloadRecordingArtifactByTicketExample
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
            var apiInstance = new RecordingsApi(httpClient, config, httpClientHandler);
            var t = "t_example";  // string | The signed ticket from the artifact-ticket endpoint.
            var recordingId = 789L;  // long | The numeric id from a recording index row.

            try
            {
                // Fetch recording media with a playback ticket
                FileParameter result = apiInstance.DownloadRecordingArtifactByTicket(t, recordingId);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling RecordingsApi.DownloadRecordingArtifactByTicket: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the DownloadRecordingArtifactByTicketWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Fetch recording media with a playback ticket
    ApiResponse<FileParameter> response = apiInstance.DownloadRecordingArtifactByTicketWithHttpInfo(t, recordingId);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling RecordingsApi.DownloadRecordingArtifactByTicketWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **t** | **string** | The signed ticket from the artifact-ticket endpoint. |  |
| **recordingId** | **long** | The numeric id from a recording index row. |  |

### Return type

[**FileParameter**](FileParameter.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: video/mp4, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The media bytes. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="downloadrecordingthumbnail"></a>
# **DownloadRecordingThumbnail**
> FileParameter DownloadRecordingThumbnail (long recordingId)

Download a recording's preview image

Owner-only. Served with `Cache-Control: private, max-age=300` because a preview is user-scoped and can change under the same URL. 

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
    public class DownloadRecordingThumbnailExample
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
            var apiInstance = new RecordingsApi(httpClient, config, httpClientHandler);
            var recordingId = 789L;  // long | The numeric id from a recording index row.

            try
            {
                // Download a recording's preview image
                FileParameter result = apiInstance.DownloadRecordingThumbnail(recordingId);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling RecordingsApi.DownloadRecordingThumbnail: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the DownloadRecordingThumbnailWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Download a recording's preview image
    ApiResponse<FileParameter> response = apiInstance.DownloadRecordingThumbnailWithHttpInfo(recordingId);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling RecordingsApi.DownloadRecordingThumbnailWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **recordingId** | **long** | The numeric id from a recording index row. |  |

### Return type

[**FileParameter**](FileParameter.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: image/jpeg, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The preview image bytes. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="downloadrecordingthumbnailbyticket"></a>
# **DownloadRecordingThumbnailByTicket**
> FileParameter DownloadRecordingThumbnailByTicket (string t, long recordingId)

Fetch a recording preview with a playback ticket

The URL returned by `POST /v1/me/recordings/{recording_id}/thumbnail-ticket`. Same ticket rules as the artifact route; an artifact ticket is rejected here (`404`) because the purpose is part of the signature. 

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
    public class DownloadRecordingThumbnailByTicketExample
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
            var apiInstance = new RecordingsApi(httpClient, config, httpClientHandler);
            var t = "t_example";  // string | The signed ticket from the thumbnail-ticket endpoint.
            var recordingId = 789L;  // long | The numeric id from a recording index row.

            try
            {
                // Fetch a recording preview with a playback ticket
                FileParameter result = apiInstance.DownloadRecordingThumbnailByTicket(t, recordingId);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling RecordingsApi.DownloadRecordingThumbnailByTicket: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the DownloadRecordingThumbnailByTicketWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Fetch a recording preview with a playback ticket
    ApiResponse<FileParameter> response = apiInstance.DownloadRecordingThumbnailByTicketWithHttpInfo(t, recordingId);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling RecordingsApi.DownloadRecordingThumbnailByTicketWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **t** | **string** | The signed ticket from the thumbnail-ticket endpoint. |  |
| **recordingId** | **long** | The numeric id from a recording index row. |  |

### Return type

[**FileParameter**](FileParameter.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: image/jpeg, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The preview image bytes. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="getsessionrecording"></a>
# **GetSessionRecording**
> SessionRecording GetSessionRecording (long recordingId)

One recording the caller owns

The same row the list returns. Another user's id is a `404` rather than a `403`, so the route cannot be used to discover which recording ids exist. 

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
    public class GetSessionRecordingExample
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
            var apiInstance = new RecordingsApi(httpClient, config, httpClientHandler);
            var recordingId = 789L;  // long | The numeric id from a recording index row.

            try
            {
                // One recording the caller owns
                SessionRecording result = apiInstance.GetSessionRecording(recordingId);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling RecordingsApi.GetSessionRecording: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the GetSessionRecordingWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // One recording the caller owns
    ApiResponse<SessionRecording> response = apiInstance.GetSessionRecordingWithHttpInfo(recordingId);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling RecordingsApi.GetSessionRecordingWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **recordingId** | **long** | The numeric id from a recording index row. |  |

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The recording. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="listsessionrecordings"></a>
# **ListSessionRecordings**
> SessionRecordingList ListSessionRecordings (string? cursor = null, int? limit = null)

The caller's session-recording index

Cursor-paginated, newest first. `next_cursor` is opaque: pass it back as `?cursor=` to fetch the next page and stop when it is absent. The envelope here has no `total` because this endpoint does not count the whole set on every page, so an integrator must page until `next_cursor` is empty rather than trusting a count. This is the metadata index only; the media bytes live behind the per-recording `.../artifact` routes documented in `docs/user-api-docs.md`. 

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
    public class ListSessionRecordingsExample
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
            var apiInstance = new RecordingsApi(httpClient, config, httpClientHandler);
            var cursor = "cursor_example";  // string? |  (optional) 
            var limit = 56;  // int? |  (optional) 

            try
            {
                // The caller's session-recording index
                SessionRecordingList result = apiInstance.ListSessionRecordings(cursor, limit);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling RecordingsApi.ListSessionRecordings: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the ListSessionRecordingsWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // The caller's session-recording index
    ApiResponse<SessionRecordingList> response = apiInstance.ListSessionRecordingsWithHttpInfo(cursor, limit);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling RecordingsApi.ListSessionRecordingsWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **cursor** | **string?** |  | [optional]  |
| **limit** | **int?** |  | [optional]  |

### Return type

[**SessionRecordingList**](SessionRecordingList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | One page of the caller&#39;s recordings, newest first. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="updatesessionrecording"></a>
# **UpdateSessionRecording**
> SessionRecording UpdateSessionRecording (long recordingId, SessionRecordingPatch sessionRecordingPatch)

Update a recording's lifecycle metadata

Partial update: only the fields present in the body are written, so a producer can patch `status` when a capture starts and then `size_bytes`/`ended_at` when the upload finishes. `status` accepts the same three values as create; anything else is a `400` and writes nothing. 

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
    public class UpdateSessionRecordingExample
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
            var apiInstance = new RecordingsApi(httpClient, config, httpClientHandler);
            var recordingId = 789L;  // long | The numeric id from a recording index row.
            var sessionRecordingPatch = new SessionRecordingPatch(); // SessionRecordingPatch | 

            try
            {
                // Update a recording's lifecycle metadata
                SessionRecording result = apiInstance.UpdateSessionRecording(recordingId, sessionRecordingPatch);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling RecordingsApi.UpdateSessionRecording: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the UpdateSessionRecordingWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Update a recording's lifecycle metadata
    ApiResponse<SessionRecording> response = apiInstance.UpdateSessionRecordingWithHttpInfo(recordingId, sessionRecordingPatch);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling RecordingsApi.UpdateSessionRecordingWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **recordingId** | **long** | The numeric id from a recording index row. |  |
| **sessionRecordingPatch** | [**SessionRecordingPatch**](SessionRecordingPatch.md) |  |  |

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The updated recording. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="uploadrecordingartifact"></a>
# **UploadRecordingArtifact**
> SessionRecording UploadRecordingArtifact (long recordingId, FileParameter file, string? ext = null)

Upload a recording's media bytes

Two request shapes, because the producers differ: `multipart/form-data` with a `file` part (the browser and Qt clients) or the raw body with `?ext=`. The extension is allow-listed (`.mp4`, `.m4v`, `.mkv`, `.webm`, `.h264`, `.grf`), the bytes are written to a temporary file and renamed into place so a dropped upload can never leave a half-written file under the real name, and the row is then updated with `artifact_path`, `artifact_url`, `format` and `size_bytes` in one step.  An empty body is a `400`; a body over the configured limit (`RECORDING_MAX_BYTES`, exposed as `store.max_bytes`) is a `400` as well, never a truncated file. Uploading again replaces the stored artifact. 

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
    public class UploadRecordingArtifactExample
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
            var apiInstance = new RecordingsApi(httpClient, config, httpClientHandler);
            var recordingId = 789L;  // long | The numeric id from a recording index row.
            var file = new System.IO.MemoryStream(System.IO.File.ReadAllBytes("/path/to/file.txt"));  // FileParameter | 
            var ext = ".mp4";  // string? | Required for the raw-body shape; ignored for multipart. (optional) 

            try
            {
                // Upload a recording's media bytes
                SessionRecording result = apiInstance.UploadRecordingArtifact(recordingId, file, ext);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling RecordingsApi.UploadRecordingArtifact: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the UploadRecordingArtifactWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Upload a recording's media bytes
    ApiResponse<SessionRecording> response = apiInstance.UploadRecordingArtifactWithHttpInfo(recordingId, file, ext);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling RecordingsApi.UploadRecordingArtifactWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **recordingId** | **long** | The numeric id from a recording index row. |  |
| **file** | **FileParameter****FileParameter** |  |  |
| **ext** | **string?** | Required for the raw-body shape; ignored for multipart. | [optional]  |

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: multipart/form-data, application/octet-stream
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The recording, now carrying the stored artifact. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

<a id="uploadrecordingthumbnail"></a>
# **UploadRecordingThumbnail**
> SessionRecording UploadRecordingThumbnail (long recordingId, FileParameter file, string? ext = null)

Upload a recording's preview image

The same two shapes as the artifact upload (`multipart/form-data` with a `file` part, or the raw body with `?ext=`), but the extension is limited to `.jpg`, `.jpeg` and `.png` and the size to 2 MiB - a preview is a small still, and an oversized one means the wrong file was sent.  The bytes are sniffed before they are stored rather than trusted from the name or the declared content type: the index renders this image inline, so a file that only claims to be a JPEG is a `400`. 

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
    public class UploadRecordingThumbnailExample
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
            var apiInstance = new RecordingsApi(httpClient, config, httpClientHandler);
            var recordingId = 789L;  // long | The numeric id from a recording index row.
            var file = new System.IO.MemoryStream(System.IO.File.ReadAllBytes("/path/to/file.txt"));  // FileParameter | 
            var ext = ".jpg";  // string? | Required for the raw-body shape; ignored for multipart. (optional) 

            try
            {
                // Upload a recording's preview image
                SessionRecording result = apiInstance.UploadRecordingThumbnail(recordingId, file, ext);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling RecordingsApi.UploadRecordingThumbnail: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the UploadRecordingThumbnailWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Upload a recording's preview image
    ApiResponse<SessionRecording> response = apiInstance.UploadRecordingThumbnailWithHttpInfo(recordingId, file, ext);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling RecordingsApi.UploadRecordingThumbnailWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **recordingId** | **long** | The numeric id from a recording index row. |  |
| **file** | **FileParameter****FileParameter** |  |  |
| **ext** | **string?** | Required for the raw-body shape; ignored for multipart. | [optional]  |

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: multipart/form-data, application/octet-stream
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The recording, now carrying the preview. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

