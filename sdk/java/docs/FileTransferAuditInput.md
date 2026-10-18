

# FileTransferAuditInput

A controller report about one file transferred outside the signaling data path. The server stores the action as `file.transfer.upload` or `file.transfer.download`, the resource as `file_transfer/{transfer_id}`, and the remaining fields as JSON details. 

## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**deviceId** | **String** | A device owned by the authenticated caller. |  |
|**transferId** | **String** | Stable id used to make retries idempotent. |  |
|**direction** | [**DirectionEnum**](#DirectionEnum) | Upload is controller to host; download is host to controller. |  |
|**status** | [**StatusEnum**](#StatusEnum) |  |  |
|**filename** | **String** |  |  [optional] |
|**bytesTransferred** | **Long** |  |  [optional] |
|**totalBytes** | **Long** | Zero means the controller did not know the total size. |  [optional] |
|**error** | **String** | Human-readable failure detail when status is failed. |  [optional] |



## Enum: DirectionEnum

| Name | Value |
|---- | -----|
| UPLOAD | &quot;upload&quot; |
| DOWNLOAD | &quot;download&quot; |



## Enum: StatusEnum

| Name | Value |
|---- | -----|
| COMPLETED | &quot;completed&quot; |
| FAILED | &quot;failed&quot; |



