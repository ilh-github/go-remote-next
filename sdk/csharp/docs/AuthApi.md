# QuickDesk.Signaling.Client.Api.AuthApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
|--------|--------------|-------------|
| [**CreateSession**](AuthApi.md#createsession) | **POST** /v1/auth/sessions | Exchange a username/password for a session |

<a id="createsession"></a>
# **CreateSession**
> AuthSessionResponse CreateSession (AuthSessionRequest authSessionRequest)

Exchange a username/password for a session

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
    public class CreateSessionExample
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
            var apiInstance = new AuthApi(httpClient, config, httpClientHandler);
            var authSessionRequest = new AuthSessionRequest(); // AuthSessionRequest | 

            try
            {
                // Exchange a username/password for a session
                AuthSessionResponse result = apiInstance.CreateSession(authSessionRequest);
                Debug.WriteLine(result);
            }
            catch (ApiException  e)
            {
                Debug.Print("Exception when calling AuthApi.CreateSession: " + e.Message);
                Debug.Print("Status Code: " + e.ErrorCode);
                Debug.Print(e.StackTrace);
            }
        }
    }
}
```

#### Using the CreateSessionWithHttpInfo variant
This returns an ApiResponse object which contains the response data, status code and headers.

```csharp
try
{
    // Exchange a username/password for a session
    ApiResponse<AuthSessionResponse> response = apiInstance.CreateSessionWithHttpInfo(authSessionRequest);
    Debug.Write("Status Code: " + response.StatusCode);
    Debug.Write("Response Headers: " + response.Headers);
    Debug.Write("Response Body: " + response.Data);
}
catch (ApiException e)
{
    Debug.Print("Exception when calling AuthApi.CreateSessionWithHttpInfo: " + e.Message);
    Debug.Print("Status Code: " + e.ErrorCode);
    Debug.Print(e.StackTrace);
}
```

### Parameters

| Name | Type | Description | Notes |
|------|------|-------------|-------|
| **authSessionRequest** | [**AuthSessionRequest**](AuthSessionRequest.md) |  |  |

### Return type

[**AuthSessionResponse**](AuthSessionResponse.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | A session with an access token and a refresh token. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **429** | Too many requests; retry after the advertised delay. |  * Retry-After - Seconds to wait before retrying. <br>  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

