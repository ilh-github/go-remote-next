# # RemoteSession

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **int** |  |
**session_id** | **string** | The signaling session id the server assigned; unique per session. |
**user_id** | **int** | The device&#39;s owner. 0 when the device was unbound at the time. |
**device_id** | **string** |  |
**client_id** | **string** | The controller-declared client id, the same value the device access log records. |
**client_ip** | **string** |  |
**started_at** | **\DateTime** |  |
**ended_at** | **\DateTime** |  |
**end_reason** | **string** | Empty while the session is open; &#x60;disconnected&#x60; when the signaling socket went away on its own, &#x60;revoked&#x60; when the owner or an administrator closed it on purpose, &#x60;server_restart&#x60; for rows a restart found still open. |
**active** | **bool** |  |
**duration_seconds** | **int** |  |
**created_at** | **\DateTime** |  |
**updated_at** | **\DateTime** |  |

[[Back to Model list]](../../README.md#models) [[Back to API list]](../../README.md#endpoints) [[Back to README]](../../README.md)
