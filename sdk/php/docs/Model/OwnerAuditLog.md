# # OwnerAuditLog

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **int** |  |
**actor_user_id** | **int** | The account that reported the transfer. Always the caller. | [optional]
**action** | **string** | &#x60;file.transfer&#x60; or &#x60;clipboard.sync&#x60;. The direction is not part of the action; it is in &#x60;details.directions&#x60;. |
**resource_type** | **string** | &#x60;file_transfer&#x60; or &#x60;clipboard_sync&#x60;. |
**resource_id** | **string** | The transfer id the reporter supplied, which is the idempotency key. |
**details** | **string** | JSON-encoded metadata: byte counts, direction, status and, for file transfers, the filename. Never the transferred content itself. | [optional]
**ip** | **string** | The source address the report arrived from. | [optional]
**created_at** | **\DateTime** |  |

[[Back to Model list]](../../README.md#models) [[Back to API list]](../../README.md#endpoints) [[Back to README]](../../README.md)
