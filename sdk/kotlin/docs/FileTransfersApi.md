# FileTransfersApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**createClipboardAudit**](FileTransfersApi.md#createClipboardAudit) | **POST** /v1/me/clipboard-audits | Report a controller-observed clipboard sync for auditing |
| [**createFileTransferAudit**](FileTransfersApi.md#createFileTransferAudit) | **POST** /v1/me/file-transfer-audits | Report a controller-observed file transfer for auditing |
| [**listMyAuditLogs**](FileTransfersApi.md#listMyAuditLogs) | **GET** /v1/me/audit-logs | Read back the data-channel audit trail that belongs to the caller |


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

<a id="listMyAuditLogs"></a>
# **listMyAuditLogs**
> OwnerAuditLogPage listMyAuditLogs(action, cursor, limit)

Read back the data-channel audit trail that belongs to the caller

The read side of the two reporting endpoints above. File and clipboard transfers travel peer to peer over the Remoting data channel, so the control plane can only record what a controller tells it afterwards; this is how the device owner reads those records back without going through the administrator console.  Rows are scoped to the caller&#39;s own actor id. Both reporting endpoints refuse a device the caller does not own, so the actor on a row is that device&#39;s owner and this scope cannot expose anybody else&#39;s transfers.  &#x60;action&#x60; is an allow-list rather than a free-form filter: an action this trail does not hold is refused with 400 rather than answered with an empty page, because \&quot;you did nothing\&quot; must not be a conclusion a typo can produce. Supported values are &#x60;file.transfer.upload&#x60;, &#x60;file.transfer.download&#x60;, &#x60;clipboard.sync.upload&#x60; and &#x60;clipboard.sync.download&#x60;.  Cursor-paginated, newest first. &#x60;next_cursor&#x60; is opaque: pass it back as &#x60;?cursor&#x3D;&#x60; to fetch the next page and stop when it is absent. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = FileTransfersApi()
val action : kotlin.String = action_example // kotlin.String | Narrow to one audit action. An unsupported value is a 400.
val cursor : kotlin.String = cursor_example // kotlin.String | 
val limit : kotlin.Int = 56 // kotlin.Int | 
try {
    val result : OwnerAuditLogPage = apiInstance.listMyAuditLogs(action, cursor, limit)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling FileTransfersApi#listMyAuditLogs")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling FileTransfersApi#listMyAuditLogs")
    e.printStackTrace()
}
```

### Parameters
| **action** | **kotlin.String**| Narrow to one audit action. An unsupported value is a 400. | [optional] [enum: file.transfer.upload, file.transfer.download, clipboard.sync.upload, clipboard.sync.download] |
| **cursor** | **kotlin.String**|  | [optional] |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **limit** | **kotlin.Int**|  | [optional] |

### Return type

[**OwnerAuditLogPage**](OwnerAuditLogPage.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

