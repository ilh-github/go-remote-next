# MeApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**getMe**](MeApi.md#getMe) | **GET** /v1/me | The authenticated user |


<a id="getMe"></a>
# **getMe**
> User getMe()

The authenticated user

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = MeApi()
try {
    val result : User = apiInstance.getMe()
    println(result)
} catch (e: ClientException) {
    println("4xx response calling MeApi#getMe")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling MeApi#getMe")
    e.printStackTrace()
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**User**](User.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

