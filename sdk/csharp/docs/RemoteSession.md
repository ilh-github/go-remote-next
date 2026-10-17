# QuickDesk.Signaling.Client.Model.RemoteSession
One controller holding, or having held, a signaling session against a device. `active` and `duration_seconds` are derived: the first from `ended_at`, the second is the run length so far while the session is still open, so a client never has to pick its own notion of \"now\". 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **long** |  | 
**SessionId** | **string** | The signaling session id the server assigned; unique per session. | 
**UserId** | **long** | The device&#39;s owner. 0 when the device was unbound at the time. | 
**DeviceId** | **string** |  | 
**ClientId** | **string** | The controller-declared client id, the same value the device access log records. | 
**ClientIp** | **string** |  | 
**StartedAt** | **DateTime** |  | 
**EndedAt** | **DateTime?** |  | 
**EndReason** | **string** | Empty while the session is open; &#x60;disconnected&#x60; when the signaling socket went away on its own, &#x60;revoked&#x60; when the owner or an administrator closed it on purpose, &#x60;server_restart&#x60; for rows a restart found still open. | 
**Active** | **bool** |  | 
**DurationSeconds** | **long** |  | 
**CreatedAt** | **DateTime** |  | 
**UpdatedAt** | **DateTime** |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

