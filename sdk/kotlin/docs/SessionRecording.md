
# SessionRecording

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **id** | **kotlin.Int** |  |  |
| **userId** | **kotlin.Int** |  |  |
| **deviceId** | **kotlin.String** |  |  |
| **deviceName** | **kotlin.String** |  |  |
| **sessionId** | **kotlin.String** |  |  |
| **status** | [**inline**](#Status) | &#x60;recording&#x60; while the capture is in progress, then &#x60;completed&#x60; or &#x60;failed&#x60;. A stale &#x60;recording&#x60; row is swept to &#x60;failed&#x60; by the recording lifecycle worker.  |  |
| **format** | **kotlin.String** | Container/codec hint, e.g. &#x60;mp4&#x60;, &#x60;h264&#x60;, &#x60;grf&#x60;. |  |
| **artifactPath** | **kotlin.String** | Server-side stored path; not a client-fetchable URL. |  |
| **artifactUrl** | **kotlin.String** |  |  |
| **thumbnailPath** | **kotlin.String** |  |  |
| **thumbnailUrl** | **kotlin.String** |  |  |
| **durationSeconds** | **kotlin.Int** |  |  |
| **sizeBytes** | **kotlin.Long** |  |  |
| **startedAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  |
| **createdAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  |
| **updatedAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  |
| **endedAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  [optional] |
| **user** | [**kotlin.Any**](.md) | The owning user, present when the server eager-loads it. Optional - clients must not depend on this being present.  |  [optional] |


<a id="Status"></a>
## Enum: status
| Name | Value |
| ---- | ----- |
| status | recording, completed, failed |



