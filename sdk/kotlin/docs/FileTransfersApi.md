# FileTransfersApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**createClipboardAudit**](FileTransfersApi.md#createClipboardAudit) | **POST** /v1/me/clipboard-audits | Report a controller-observed clipboard sync for auditing |
| [**createFileTransferAudit**](FileTransfersApi.md#createFileTransferAudit) | **POST** /v1/me/file-transfer-audits | Report a controller-observed file transfer for auditing |


<a id="createClipboardAudit"></a>
# **createClipboardAudit**
> createClipboardAudit(clipboardAuditInput)

Report a controller-observed clipboard sync for auditing

Clipboard bytes travel directly over the Remoting data channel and never pass through the signaling server. This endpoint lets an authenticated device owner report a clipboard sync after the fact so it appears in the audit trail.  The report carries **metadata only**: direction, MIME type and byte count. There is deliberately no field for the clipboard text. A clipboard is by definition whatever the user last copied - routinely a password, a recovery code or a private message - so a row that stored the text would turn the audit trail into a credential store every later reader of the database inherits. \&quot;The operator copied 1.2 KB of text at 14:03\&quot; answers the investigative question; the text answers nothing an investigator needs and costs everything if the database leaks.  Reports are idempotent by &#x60;transfer_id&#x60;, and the device must belong to the caller. This is an attributable controller report, not cryptographic proof from the host that a clipboard sync happened. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = FileTransfersApi()
val clipboardAuditInput : ClipboardAuditInput =  // ClipboardAuditInput | 
try {
    apiInstance.createClipboardAudit(clipboardAuditInput)
} catch (e: ClientException) {
    println("4xx response calling FileTransfersApi#createClipboardAudit")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling FileTransfersApi#createClipboardAudit")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **clipboardAuditInput** | [**ClipboardAuditInput**](ClipboardAuditInput.md)|  | |

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

