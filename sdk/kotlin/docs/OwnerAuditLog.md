
# OwnerAuditLog

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **id** | **kotlin.Long** |  |  |
| **action** | **kotlin.String** | One of &#x60;file.transfer.upload&#x60;, &#x60;file.transfer.download&#x60;, &#x60;clipboard.sync.upload&#x60;, &#x60;clipboard.sync.download&#x60;.  |  |
| **resourceType** | **kotlin.String** | &#x60;file_transfer&#x60; or &#x60;clipboard_sync&#x60;.  |  |
| **resourceId** | **kotlin.String** | The transfer id the reporter supplied, which is the idempotency key. |  |
| **createdAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  |
| **actorUserId** | **kotlin.Long** | The account that reported the transfer. Always the caller. |  [optional] |
| **details** | **kotlin.String** | JSON-encoded metadata: byte counts, direction, status and, for file transfers, the filename. Never the transferred content itself.  |  [optional] |
| **ip** | **kotlin.String** | The source address the report arrived from. |  [optional] |



