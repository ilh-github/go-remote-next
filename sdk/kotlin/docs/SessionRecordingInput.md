
# SessionRecordingInput

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **deviceId** | **kotlin.String** | The device the capture came from. |  |
| **status** | [**inline**](#Status) | How far the capture got when this row was written. |  |
| **deviceName** | **kotlin.String** |  |  [optional] |
| **sessionId** | **kotlin.String** | The control session this recording belongs to, when there is one. |  [optional] |
| **format** | **kotlin.String** | Free-form label; the upload route overwrites it from the file extension. |  [optional] |
| **artifactPath** | **kotlin.String** | Server-side path. Set by the upload route; clients do not need to send it. |  [optional] |
| **artifactUrl** | **kotlin.String** |  |  [optional] |
| **thumbnailPath** | **kotlin.String** |  |  [optional] |
| **durationSeconds** | **kotlin.Int** |  |  [optional] |
| **sizeBytes** | **kotlin.Int** |  |  [optional] |
| **startedAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) | Defaults to the time the row was created when omitted. |  [optional] |
| **endedAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  [optional] |


<a id="Status"></a>
## Enum: status
| Name | Value |
| ---- | ----- |
| status | recording, completed, failed |



