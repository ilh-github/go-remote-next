
# ClipboardAuditInput

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **deviceId** | **kotlin.String** | A device owned by the authenticated caller. |  |
| **transferId** | **kotlin.String** | Stable id used to make retries idempotent. |  |
| **direction** | [**inline**](#Direction) | Which way the clipboard went: upload is controller to host, download is host to controller. |  |
| **status** | [**inline**](#Status) |  |  |
| **mimeType** | **kotlin.String** | The clipboard format, e.g. &#x60;text/plain&#x60;. Metadata only. |  |
| **bytes** | **kotlin.Long** | Size of the payload that crossed the data channel. |  [optional] |


<a id="Direction"></a>
## Enum: direction
| Name | Value |
| ---- | ----- |
| direction | upload, download |


<a id="Status"></a>
## Enum: status
| Name | Value |
| ---- | ----- |
| status | completed, failed |



