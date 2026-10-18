# FileTransfersApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**createFileTransferAudit**](FileTransfersApi.md#createFileTransferAudit) | **POST** /v1/me/file-transfer-audits | Report a controller-observed file transfer for auditing |


<a id="createFileTransferAudit"></a>
# **createFileTransferAudit**
> createFileTransferAudit(fileTransferAuditInput)

Report a controller-observed file transfer for auditing

File bytes travel directly over the Remoting data channel and never pass through the signaling server. This endpoint lets an authenticated device owner report a completed or failed transfer after the fact so it appears in the audit trail.  Reports are idempotent by &#x60;(actor, direction, transfer_id)&#x60;: retrying the same event does not create another row. The device must belong to the caller. This is an attributable controller report, not cryptographic proof from the host that a file operation happened. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = FileTransfersApi()
val fileTransferAuditInput : FileTransferAuditInput =  // FileTransferAuditInput | 
try {
    apiInstance.createFileTransferAudit(fileTransferAuditInput)
} catch (e: ClientException) {
    println("4xx response calling FileTransfersApi#createFileTransferAudit")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling FileTransfersApi#createFileTransferAudit")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **fileTransferAuditInput** | [**FileTransferAuditInput**](FileTransferAuditInput.md)|  | |

### Return type

null (empty response body)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/problem+json

