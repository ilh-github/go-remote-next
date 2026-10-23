# # ClipboardAuditInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**device_id** | **string** | A device owned by the authenticated caller. |
**transfer_id** | **string** | Stable id used to make retries idempotent. |
**direction** | **string** | Which way the clipboard went: upload is controller to host, download is host to controller. |
**status** | **string** |  |
**mime_type** | **string** | The clipboard format, e.g. &#x60;text/plain&#x60;. Metadata only. |
**bytes** | **int** | Size of the payload that crossed the data channel. | [optional]

[[Back to Model list]](../../README.md#models) [[Back to API list]](../../README.md#endpoints) [[Back to README]](../../README.md)
