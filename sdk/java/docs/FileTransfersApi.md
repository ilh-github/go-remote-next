# FileTransfersApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**createClipboardAudit**](FileTransfersApi.md#createClipboardAudit) | **POST** /v1/me/clipboard-audits | Report a controller-observed clipboard sync for auditing |
| [**createFileTransferAudit**](FileTransfersApi.md#createFileTransferAudit) | **POST** /v1/me/file-transfer-audits | Report a controller-observed file transfer for auditing |
| [**listMyAuditLogs**](FileTransfersApi.md#listMyAuditLogs) | **GET** /v1/me/audit-logs | Read back the data-channel audit trail that belongs to the caller |


<a id="createClipboardAudit"></a>
# **createClipboardAudit**
> createClipboardAudit(clipboardAuditInput)

Report a controller-observed clipboard sync for auditing

Clipboard bytes travel directly over the Remoting data channel and never pass through the signaling server. This endpoint lets an authenticated device owner report a clipboard sync after the fact so it appears in the audit trail.  The report carries **metadata only**: direction, MIME type and byte count. There is deliberately no field for the clipboard text. A clipboard is by definition whatever the user last copied - routinely a password, a recovery code or a private message - so a row that stored the text would turn the audit trail into a credential store every later reader of the database inherits. \&quot;The operator copied 1.2 KB of text at 14:03\&quot; answers the investigative question; the text answers nothing an investigator needs and costs everything if the database leaks.  Reports are idempotent by &#x60;transfer_id&#x60;, and the device must belong to the caller. This is an attributable controller report, not cryptographic proof from the host that a clipboard sync happened. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.FileTransfersApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("https://your-signaling-host");
    
    // Configure API key authorization: ApiKeyAuth
    ApiKeyAuth ApiKeyAuth = (ApiKeyAuth) defaultClient.getAuthentication("ApiKeyAuth");
    ApiKeyAuth.setApiKey("YOUR API KEY");
    // Uncomment the following line to set a prefix for the API key, e.g. "Token" (defaults to null)
    //ApiKeyAuth.setApiKeyPrefix("Token");

    // Configure HTTP bearer authorization: BearerAuth
    HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
    BearerAuth.setBearerToken("BEARER TOKEN");

    FileTransfersApi apiInstance = new FileTransfersApi(defaultClient);
    ClipboardAuditInput clipboardAuditInput = new ClipboardAuditInput(); // ClipboardAuditInput | 
    try {
      apiInstance.createClipboardAudit(clipboardAuditInput);
    } catch (ApiException e) {
      System.err.println("Exception when calling FileTransfersApi#createClipboardAudit");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters

| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **clipboardAuditInput** | [**ClipboardAuditInput**](ClipboardAuditInput.md)|  | |

### Return type

null (empty response body)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **204** | The clipboard report is in the audit trail (or was already recorded). |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="createFileTransferAudit"></a>
# **createFileTransferAudit**
> createFileTransferAudit(fileTransferAuditInput)

Report a controller-observed file transfer for auditing

File bytes travel directly over the Remoting data channel and never pass through the signaling server. This endpoint lets an authenticated device owner report a completed or failed transfer after the fact so it appears in the audit trail.  Reports are idempotent by &#x60;(actor, direction, transfer_id)&#x60;: retrying the same event does not create another row. The device must belong to the caller. This is an attributable controller report, not cryptographic proof from the host that a file operation happened. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.FileTransfersApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("https://your-signaling-host");
    
    // Configure API key authorization: ApiKeyAuth
    ApiKeyAuth ApiKeyAuth = (ApiKeyAuth) defaultClient.getAuthentication("ApiKeyAuth");
    ApiKeyAuth.setApiKey("YOUR API KEY");
    // Uncomment the following line to set a prefix for the API key, e.g. "Token" (defaults to null)
    //ApiKeyAuth.setApiKeyPrefix("Token");

    // Configure HTTP bearer authorization: BearerAuth
    HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
    BearerAuth.setBearerToken("BEARER TOKEN");

    FileTransfersApi apiInstance = new FileTransfersApi(defaultClient);
    FileTransferAuditInput fileTransferAuditInput = new FileTransferAuditInput(); // FileTransferAuditInput | 
    try {
      apiInstance.createFileTransferAudit(fileTransferAuditInput);
    } catch (ApiException e) {
      System.err.println("Exception when calling FileTransfersApi#createFileTransferAudit");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters

| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **fileTransferAuditInput** | [**FileTransferAuditInput**](FileTransferAuditInput.md)|  | |

### Return type

null (empty response body)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **204** | The transfer report is in the audit trail (or was already recorded). |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="listMyAuditLogs"></a>
# **listMyAuditLogs**
> OwnerAuditLogPage listMyAuditLogs(action, cursor, limit)

Read back the data-channel audit trail that belongs to the caller

The read side of the two reporting endpoints above. File and clipboard transfers travel peer to peer over the Remoting data channel, so the control plane can only record what a controller tells it afterwards; this is how the device owner reads those records back without going through the administrator console.  Rows are scoped to the caller&#39;s own actor id. Both reporting endpoints refuse a device the caller does not own, so the actor on a row is that device&#39;s owner and this scope cannot expose anybody else&#39;s transfers.  &#x60;action&#x60; is an allow-list rather than a free-form filter: an action this trail does not hold is refused with 400 rather than answered with an empty page, because \&quot;you did nothing\&quot; must not be a conclusion a typo can produce. Supported values are &#x60;file.transfer&#x60; and &#x60;clipboard.sync&#x60; (one row per transfer, whichever way it went) plus the direction-scoped &#x60;file.transfer.upload&#x60;, &#x60;file.transfer.download&#x60;, &#x60;clipboard.sync.upload&#x60; and &#x60;clipboard.sync.download&#x60;.  The direction is not part of a row&#39;s identity: a transfer id reported with both directions is one row whose &#x60;details.directions&#x60; lists both. The direction-scoped filters still work - they resolve to the aggregate action plus that direction - because \&quot;what did I upload\&quot; is a real question.  Cursor-paginated, newest first. &#x60;next_cursor&#x60; is opaque: pass it back as &#x60;?cursor&#x3D;&#x60; to fetch the next page and stop when it is absent. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.FileTransfersApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("https://your-signaling-host");
    
    // Configure API key authorization: ApiKeyAuth
    ApiKeyAuth ApiKeyAuth = (ApiKeyAuth) defaultClient.getAuthentication("ApiKeyAuth");
    ApiKeyAuth.setApiKey("YOUR API KEY");
    // Uncomment the following line to set a prefix for the API key, e.g. "Token" (defaults to null)
    //ApiKeyAuth.setApiKeyPrefix("Token");

    // Configure HTTP bearer authorization: BearerAuth
    HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
    BearerAuth.setBearerToken("BEARER TOKEN");

    FileTransfersApi apiInstance = new FileTransfersApi(defaultClient);
    String action = "file.transfer"; // String | Narrow to one audit action. An unsupported value is a 400. The aggregate names return one row per transfer; the direction-scoped names return the rows that went that way. 
    String cursor = "cursor_example"; // String | 
    Integer limit = 56; // Integer | 
    try {
      OwnerAuditLogPage result = apiInstance.listMyAuditLogs(action, cursor, limit);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling FileTransfersApi#listMyAuditLogs");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters

| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **action** | **String**| Narrow to one audit action. An unsupported value is a 400. The aggregate names return one row per transfer; the direction-scoped names return the rows that went that way.  | [optional] [enum: file.transfer, clipboard.sync, file.transfer.upload, file.transfer.download, clipboard.sync.upload, clipboard.sync.download] |
| **cursor** | **String**|  | [optional] |
| **limit** | **Integer**|  | [optional] |

### Return type

[**OwnerAuditLogPage**](OwnerAuditLogPage.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | One page of the caller&#39;s audit rows, newest first. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |

