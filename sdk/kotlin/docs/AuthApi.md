# AuthApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**createSession**](AuthApi.md#createSession) | **POST** /v1/auth/sessions | Exchange a username/password for a session |


<a id="createSession"></a>
# **createSession**
> AuthSessionResponse createSession(authSessionRequest)

Exchange a username/password for a session

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = AuthApi()
val authSessionRequest : AuthSessionRequest =  // AuthSessionRequest | 
try {
    val result : AuthSessionResponse = apiInstance.createSession(authSessionRequest)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling AuthApi#createSession")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling AuthApi#createSession")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **authSessionRequest** | [**AuthSessionRequest**](AuthSessionRequest.md)|  | |

### Return type

[**AuthSessionResponse**](AuthSessionResponse.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

