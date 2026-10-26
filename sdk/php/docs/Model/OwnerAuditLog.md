# # OwnerAuditLog

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **int** |  |
**actor_user_id** | **int** | The account that reported the transfer. Always the caller. | [optional]
**action** | **string** | One of &#x60;file.transfer.upload&#x60;, &#x60;file.transfer.download&#x60;, &#x60;clipboard.sync.upload&#x60;, &#x60;clipboard.sync.download&#x60;. |
**resource_type** | **string** | &#x60;file_transfer&#x60; or &#x60;clipboard_sync&#x60;. |
**resource_id** | **string** | The transfer id the reporter supplied, which is the idempotency key. |
**details** | **string** | JSON-encoded metadata: byte counts, direction, status and, for file transfers, the filename. Never the transferred content itself. | [optional]
**ip** | **string** | The source address the report arrived from. | [optional]
**created_at** | **\DateTime** |  |

[[Back to Model list]](../../README.md#models) [[Back to API list]](../../README.md#endpoints) [[Back to README]](../../README.md)
