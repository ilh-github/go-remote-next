
# OwnerAuditLog

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **id** | **kotlin.Long** |  |  |
| **action** | **kotlin.String** | &#x60;file.transfer&#x60; or &#x60;clipboard.sync&#x60;. The direction is not part of the action; it is in &#x60;details.directions&#x60;.  |  |
| **resourceType** | **kotlin.String** | &#x60;file_transfer&#x60; or &#x60;clipboard_sync&#x60;.  |  |
| **resourceId** | **kotlin.String** | The transfer id the reporter supplied, which is the idempotency key. |  |
| **createdAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  |
| **actorUserId** | **kotlin.Long** | The account that reported the transfer. Always the caller. |  [optional] |
| **details** | **kotlin.String** | JSON-encoded metadata: byte counts, direction, status and, for file transfers, the filename. Never the transferred content itself.  |  [optional] |
| **ip** | **kotlin.String** | The source address the report arrived from. |  [optional] |



