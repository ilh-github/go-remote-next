
# FileTransferAuditInput

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **deviceId** | **kotlin.String** | A device owned by the authenticated caller. |  |
| **transferId** | **kotlin.String** | Stable id used to make retries idempotent. |  |
| **direction** | [**inline**](#Direction) | Upload is controller to host; download is host to controller. |  |
| **status** | [**inline**](#Status) |  |  |
| **filename** | **kotlin.String** |  |  [optional] |
| **bytesTransferred** | **kotlin.Long** |  |  [optional] |
| **totalBytes** | **kotlin.Long** | Zero means the controller did not know the total size. |  [optional] |
| **error** | **kotlin.String** | Human-readable failure detail when status is failed. |  [optional] |


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



