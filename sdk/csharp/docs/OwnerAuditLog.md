# QuickDesk.Signaling.Client.Model.OwnerAuditLog
One audit row. `admin_id` and `admin_username` are always empty here: this trail is written by end-user actors, and an administrator acting on the same resource lands in the admin-side trail instead. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **long** |  | 
**ActorUserId** | **long** | The account that reported the transfer. Always the caller. | [optional] 
**Action** | **string** | &#x60;file.transfer&#x60; or &#x60;clipboard.sync&#x60;. The direction is not part of the action; it is in &#x60;details.directions&#x60;.  | 
**ResourceType** | **string** | &#x60;file_transfer&#x60; or &#x60;clipboard_sync&#x60;.  | 
**ResourceId** | **string** | The transfer id the reporter supplied, which is the idempotency key. | 
**Details** | **string** | JSON-encoded metadata: byte counts, direction, status and, for file transfers, the filename. Never the transferred content itself.  | [optional] 
**Ip** | **string** | The source address the report arrived from. | [optional] 
**CreatedAt** | **DateTime** |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

