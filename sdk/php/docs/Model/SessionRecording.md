# # SessionRecording

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **int** |  |
**user_id** | **int** |  |
**device_id** | **string** |  |
**device_name** | **string** |  |
**session_id** | **string** |  |
**status** | **string** | &#x60;recording&#x60; while the capture is in progress, then &#x60;completed&#x60; or &#x60;failed&#x60;. A stale &#x60;recording&#x60; row is swept to &#x60;failed&#x60; by the recording lifecycle worker. |
**format** | **string** | Container/codec hint, e.g. &#x60;mp4&#x60;, &#x60;h264&#x60;, &#x60;grf&#x60;. |
**artifact_path** | **string** | Server-side stored path; not a client-fetchable URL. |
**artifact_url** | **string** |  |
**thumbnail_path** | **string** |  |
**thumbnail_url** | **string** |  |
**duration_seconds** | **int** |  |
**size_bytes** | **int** |  |
**started_at** | **\DateTime** |  |
**ended_at** | **\DateTime** |  | [optional]
**created_at** | **\DateTime** |  |
**updated_at** | **\DateTime** |  |
**user** | **object** | The owning user, present when the server eager-loads it. Optional - clients must not depend on this being present. | [optional]

[[Back to Model list]](../../README.md#models) [[Back to API list]](../../README.md#endpoints) [[Back to README]](../../README.md)
