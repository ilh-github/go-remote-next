# # FileTransferAuditInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**device_id** | **string** | A device owned by the authenticated caller. |
**transfer_id** | **string** | Stable id used to make retries idempotent. |
**direction** | **string** | Upload is controller to host; download is host to controller. |
**status** | **string** |  |
**filename** | **string** |  | [optional]
**bytes_transferred** | **int** |  | [optional]
**total_bytes** | **int** | Zero means the controller did not know the total size. | [optional]
**error** | **string** | Human-readable failure detail when status is failed. | [optional]

[[Back to Model list]](../../README.md#models) [[Back to API list]](../../README.md#endpoints) [[Back to README]](../../README.md)
