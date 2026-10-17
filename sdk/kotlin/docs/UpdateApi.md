# UpdateApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**checkForUpdate**](UpdateApi.md#checkForUpdate) | **GET** /v1/update:check | Check the self-hosted update channel for a newer build |
| [**downloadUpdateArtifact**](UpdateApi.md#downloadUpdateArtifact) | **GET** /v1/update-releases/{release_id}/artifact | Download an installer this server hosts itself |


<a id="checkForUpdate"></a>
# **checkForUpdate**
> UpdateCheckResult checkForUpdate(platform, arch, channel, currentVersion)

Check the self-hosted update channel for a newer build

The client-facing half of the self-hosted update channel. It is API-key gated but not user-authenticated: a controlled machine checks for updates before anyone logs in.  The server owns the index of published releases. The artifact lives at &#x60;download_url&#x60;: normally a host the operator controls, or this server itself when the installer was uploaded to it (then &#x60;download_url&#x60; is &#x60;GET /v1/update-releases/{release_id}/artifact&#x60; and &#x60;hosted_here&#x60; is true). The index is managed through the admin API (&#x60;POST /v1/admin/update-releases&#x60; and friends), which stays outside this business contract.  &#x60;current_version&#x60; is optional. Without it the newest release is returned and &#x60;update_available&#x60; is true whenever one exists; with it, &#x60;update_available&#x60; is true only when the published version is strictly newer, and &#x60;mandatory&#x60; mirrors the release flag once an update is available. &#x60;latest&#x60; is always present and is &#x60;null&#x60; when the channel has no release for that platform/arch. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = UpdateApi()
val platform : kotlin.String = platform_example // kotlin.String | Which operating system the caller is running.
val arch : kotlin.String = arch_example // kotlin.String | Which architecture the caller is running.
val channel : kotlin.String = channel_example // kotlin.String | Defaults to `stable`.
val currentVersion : kotlin.String = currentVersion_example // kotlin.String | The dotted numeric version the caller is running, e.g. `4.2.5.0`. A leading `v` is accepted.
try {
    val result : UpdateCheckResult = apiInstance.checkForUpdate(platform, arch, channel, currentVersion)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling UpdateApi#checkForUpdate")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling UpdateApi#checkForUpdate")
    e.printStackTrace()
}
```

### Parameters
| **platform** | **kotlin.String**| Which operating system the caller is running. | [enum: windows, macos, linux] |
| **arch** | **kotlin.String**| Which architecture the caller is running. | [enum: x64, arm64, universal] |
| **channel** | **kotlin.String**| Defaults to &#x60;stable&#x60;. | [optional] [enum: stable, beta, nightly] |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **currentVersion** | **kotlin.String**| The dotted numeric version the caller is running, e.g. &#x60;4.2.5.0&#x60;. A leading &#x60;v&#x60; is accepted. | [optional] |

### Return type

[**UpdateCheckResult**](UpdateCheckResult.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

<a id="downloadUpdateArtifact"></a>
# **downloadUpdateArtifact**
> java.io.File downloadUpdateArtifact(releaseId)

Download an installer this server hosts itself

The other half of a re-hosted release. &#x60;GET /v1/update:check&#x60; returns the &#x60;download_url&#x60; of the newest build, and for a release whose bytes were uploaded to this server that URL is this route, under the same API key the client already used for the check.  API-key gated but not user-authenticated, like the check itself. The response supports range requests, so a transfer that drops can resume.  A release whose bytes live on an external static host is not served here: fetching its route is a &#x60;404&#x60;, and the client should use the &#x60;download_url&#x60; from the check response instead. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = UpdateApi()
val releaseId : kotlin.Long = 789 // kotlin.Long | The `id` of the release from the check response.
try {
    val result : java.io.File = apiInstance.downloadUpdateArtifact(releaseId)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling UpdateApi#downloadUpdateArtifact")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling UpdateApi#downloadUpdateArtifact")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **releaseId** | **kotlin.Long**| The &#x60;id&#x60; of the release from the check response. | |

### Return type

[**java.io.File**](java.io.File.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/octet-stream, application/problem+json

