# RemoteSessionsApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**getRemoteSession**](RemoteSessionsApi.md#getRemoteSession) | **GET** /v1/me/remote-sessions/{session_id} | One control session with the records that share its id |
| [**listDeviceRemoteSessions**](RemoteSessionsApi.md#listDeviceRemoteSessions) | **GET** /v1/me/devices/{device_id}/sessions | One device&#39;s control sessions, each with its related records |
| [**listRemoteSessions**](RemoteSessionsApi.md#listRemoteSessions) | **GET** /v1/me/remote-sessions | The caller&#39;s control sessions, open and finished |
| [**revokeDeviceSessions**](RemoteSessionsApi.md#revokeDeviceSessions) | **POST** /v1/me/devices/{device_id}/sessions:revoke | End every live control session on one device |
| [**revokeRemoteSession**](RemoteSessionsApi.md#revokeRemoteSession) | **POST** /v1/me/remote-sessions/{session_id}/revoke | End one of the caller&#39;s control sessions |


<a id="getRemoteSession"></a>
# **getRemoteSession**
> RemoteSessionDetail getRemoteSession(sessionId)

One control session with the records that share its id

What happened in one session: the session row itself, the recordings produced during it, and the device access-log entries that carry its &#x60;session_id&#x60; (in practice the &#x60;session_started&#x60; row written when the controller authenticated). The three tables always existed - this is the read that puts them back together.  Owner only. A session belonging to another account is reported as &#x60;404&#x60; rather than &#x60;403&#x60;, so a caller cannot probe which session ids exist. Both lists are capped at 100 entries and are empty arrays rather than absent when there is nothing to show. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = RemoteSessionsApi()
val sessionId : kotlin.String = sessionId_example // kotlin.String | The signaling session id from the list.
try {
    val result : RemoteSessionDetail = apiInstance.getRemoteSession(sessionId)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling RemoteSessionsApi#getRemoteSession")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling RemoteSessionsApi#getRemoteSession")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **sessionId** | **kotlin.String**| The signaling session id from the list. | |

### Return type

[**RemoteSessionDetail**](RemoteSessionDetail.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

<a id="listDeviceRemoteSessions"></a>
# **listDeviceRemoteSessions**
> RemoteSessionDetailList listDeviceRemoteSessions(deviceId, status, cursor, limit)

One device&#39;s control sessions, each with its related records

The same history as &#x60;GET /v1/me/remote-sessions?device_id&#x3D;&#x60;, but every entry arrives already joined with the records that share its session id - the shape a device page needs, so a client never has to know a session id (or make a second request per row) to reach the join.  &#x60;status&#x60; is &#x60;active&#x60;, &#x60;ended&#x60; or omitted/&#x60;all&#x60;; &#x60;cursor&#x60;/&#x60;limit&#x60; page it newest first like every other list here. The related arrays are batched across the page, so a page costs three queries rather than three per row, and they are present-but-empty rather than absent.  A device the caller does not own is &#x60;403&#x60; and an unknown device is &#x60;404&#x60;, matching &#x60;.../access-logs&#x60;. (The session detail read answers &#x60;404&#x60; for somebody else&#39;s session instead, because a session id is looked up rather than named by its owner.) 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = RemoteSessionsApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
val status : kotlin.String = status_example // kotlin.String | Defaults to `all`.
val cursor : kotlin.String = cursor_example // kotlin.String | 
val limit : kotlin.Int = 56 // kotlin.Int | 
try {
    val result : RemoteSessionDetailList = apiInstance.listDeviceRemoteSessions(deviceId, status, cursor, limit)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling RemoteSessionsApi#listDeviceRemoteSessions")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling RemoteSessionsApi#listDeviceRemoteSessions")
    e.printStackTrace()
}
```

### Parameters
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |
| **status** | **kotlin.String**| Defaults to &#x60;all&#x60;. | [optional] [enum: active, ended, all] |
| **cursor** | **kotlin.String**|  | [optional] |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **limit** | **kotlin.Int**|  | [optional] |

### Return type

[**RemoteSessionDetailList**](RemoteSessionDetailList.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

<a id="listRemoteSessions"></a>
# **listRemoteSessions**
> RemoteSessionList listRemoteSessions(status, deviceId, cursor, limit)

The caller&#39;s control sessions, open and finished

Who is in the caller&#39;s machines right now, and who has been.  A remote session is one controller holding a signaling session against a device. It is opened when that controller&#39;s signaling socket completes authentication and closed when that socket ends, which is the whole lifetime this server can observe: the media path is peer to peer, so a session whose signaling channel is gone is over as far as the control plane is concerned.  &#x60;status&#x3D;active&#x60; answers \&quot;who is in right now\&quot;, &#x60;ended&#x60; is history, and the default lists both. &#x60;device_id&#x60; narrows to one machine. Cursor-paginated, newest first, &#x60;next_cursor&#x60; absent on the last page. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = RemoteSessionsApi()
val status : kotlin.String = status_example // kotlin.String | Defaults to `all`.
val deviceId : kotlin.String = deviceId_example // kotlin.String | 
val cursor : kotlin.String = cursor_example // kotlin.String | 
val limit : kotlin.Int = 56 // kotlin.Int | 
try {
    val result : RemoteSessionList = apiInstance.listRemoteSessions(status, deviceId, cursor, limit)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling RemoteSessionsApi#listRemoteSessions")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling RemoteSessionsApi#listRemoteSessions")
    e.printStackTrace()
}
```

### Parameters
| **status** | **kotlin.String**| Defaults to &#x60;all&#x60;. | [optional] [enum: active, ended, all] |
| **deviceId** | **kotlin.String**|  | [optional] |
| **cursor** | **kotlin.String**|  | [optional] |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **limit** | **kotlin.Int**|  | [optional] |

### Return type

[**RemoteSessionList**](RemoteSessionList.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

<a id="revokeDeviceSessions"></a>
# **revokeDeviceSessions**
> RemoteSessionBatchRevocation revokeDeviceSessions(deviceId)

End every live control session on one device

The owner&#39;s \&quot;throw everyone out of this machine\&quot;: every live controller on the device is ended in one call, whichever server instance happens to hold each socket. Each is closed at its owning instance, and the call confirms the database row before reporting it - the same promise the single-session revoke makes.  A batch does not fail as a whole: one session that cannot be reached must not stop the others, so the response reports each session&#39;s outcome and counts. &#x60;truncated&#x60; says the device had more live controllers than one call handles (the cap is 200), which means the machine is not necessarily empty when the call returns. A machine with nothing live is a &#x60;200&#x60; with &#x60;revoked: 0&#x60;, not an error.  Ownership answers match the sibling routes: &#x60;404&#x60; for a device that does not exist, &#x60;403&#x60; for one that belongs to somebody else. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = RemoteSessionsApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
try {
    val result : RemoteSessionBatchRevocation = apiInstance.revokeDeviceSessions(deviceId)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling RemoteSessionsApi#revokeDeviceSessions")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling RemoteSessionsApi#revokeDeviceSessions")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |

### Return type

[**RemoteSessionBatchRevocation**](RemoteSessionBatchRevocation.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

<a id="revokeRemoteSession"></a>
# **revokeRemoteSession**
> RemoteSessionRevocation revokeRemoteSession(sessionId)

End one of the caller&#39;s control sessions

The owner&#39;s \&quot;throw them out\&quot;. A remote session lives exactly as long as the controller&#39;s signaling socket, so ending the session means closing that socket; the session row is stamped &#x60;end_reason: revoked&#x60; and the owner&#39;s realtime stream carries &#x60;remote_session.ended&#x60; with that reason. Closing the socket also frees the device&#39;s controller slot, so a revoked session stops counting against the access policy&#39;s &#x60;max_sessions&#x60;.  Owner only, with the same concealment as the detail read: a session that belongs to another account is a &#x60;404&#x60;, not a &#x60;403&#x60;. The socket may be held by a different server instance - the row records which one admitted it - so the request is addressed at that instance and this call waits for the session to actually close: a &#x60;200&#x60; means it happened, not that a command was merely published. A session that is already over, or whose owning instance did not close it in time, is a &#x60;409&#x60;, because the one thing worse than refusing a revoke is reporting a success that did not happen. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = RemoteSessionsApi()
val sessionId : kotlin.String = sessionId_example // kotlin.String | The signaling session id from the list.
try {
    val result : RemoteSessionRevocation = apiInstance.revokeRemoteSession(sessionId)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling RemoteSessionsApi#revokeRemoteSession")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling RemoteSessionsApi#revokeRemoteSession")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **sessionId** | **kotlin.String**| The signaling session id from the list. | |

### Return type

[**RemoteSessionRevocation**](RemoteSessionRevocation.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

