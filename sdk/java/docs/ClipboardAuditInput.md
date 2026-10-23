

# ClipboardAuditInput

A controller report about one clipboard sync that happened outside the signaling data path. The server stores the action as `clipboard.sync.upload` or `clipboard.sync.download`, the resource as `clipboard_sync/{transfer_id}`, and the fields below as JSON details.  There is intentionally no field for the clipboard text. See the endpoint description for why that is a property of the contract rather than an omission from it. 

## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**deviceId** | **String** | A device owned by the authenticated caller. |  |
|**transferId** | **String** | Stable id used to make retries idempotent. |  |
|**direction** | [**DirectionEnum**](#DirectionEnum) | Which way the clipboard went: upload is controller to host, download is host to controller. |  |
|**status** | [**StatusEnum**](#StatusEnum) |  |  |
|**mimeType** | **String** | The clipboard format, e.g. &#x60;text/plain&#x60;. Metadata only. |  |
|**bytes** | **Long** | Size of the payload that crossed the data channel. |  [optional] |



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



