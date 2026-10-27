

# OwnerAuditLog

One audit row. `admin_id` and `admin_username` are always empty here: this trail is written by end-user actors, and an administrator acting on the same resource lands in the admin-side trail instead. 

## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**id** | **Long** |  |  |
|**actorUserId** | **Long** | The account that reported the transfer. Always the caller. |  [optional] |
|**action** | **String** | &#x60;file.transfer&#x60; or &#x60;clipboard.sync&#x60;. The direction is not part of the action; it is in &#x60;details.directions&#x60;.  |  |
|**resourceType** | **String** | &#x60;file_transfer&#x60; or &#x60;clipboard_sync&#x60;.  |  |
|**resourceId** | **String** | The transfer id the reporter supplied, which is the idempotency key. |  |
|**details** | **String** | JSON-encoded metadata: byte counts, direction, status and, for file transfers, the filename. Never the transferred content itself.  |  [optional] |
|**ip** | **String** | The source address the report arrived from. |  [optional] |
|**createdAt** | **OffsetDateTime** |  |  |



