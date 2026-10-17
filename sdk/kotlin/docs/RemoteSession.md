
# RemoteSession

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **id** | **kotlin.Long** |  |  |
| **sessionId** | **kotlin.String** | The signaling session id the server assigned; unique per session. |  |
| **userId** | **kotlin.Long** | The device&#39;s owner. 0 when the device was unbound at the time. |  |
| **deviceId** | **kotlin.String** |  |  |
| **clientId** | **kotlin.String** | The controller-declared client id, the same value the device access log records. |  |
| **clientIp** | **kotlin.String** |  |  |
| **startedAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  |
| **endedAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  |
| **endReason** | **kotlin.String** | Empty while the session is open; &#x60;disconnected&#x60; when the signaling socket went away on its own, &#x60;revoked&#x60; when the owner or an administrator closed it on purpose, &#x60;server_restart&#x60; for rows a restart found still open. |  |
| **active** | **kotlin.Boolean** |  |  |
| **durationSeconds** | **kotlin.Long** |  |  |
| **createdAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  |
| **updatedAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  |



