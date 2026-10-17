# QuickDesk.Signaling.Client.Model.SessionRecording
One indexed recording. The row is the control-plane metadata; the media artifact is produced by the control engine and fetched through the artifact routes. `artifact_path` is a server-side path and is not a download URL - clients should call `.../artifact-ticket` and use the URL they get back. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **int** |  | 
**UserId** | **int** |  | 
**DeviceId** | **string** |  | 
**DeviceName** | **string** |  | 
**SessionId** | **string** |  | 
**Status** | **string** | &#x60;recording&#x60; while the capture is in progress, then &#x60;completed&#x60; or &#x60;failed&#x60;. A stale &#x60;recording&#x60; row is swept to &#x60;failed&#x60; by the recording lifecycle worker.  | 
**Format** | **string** | Container/codec hint, e.g. &#x60;mp4&#x60;, &#x60;h264&#x60;, &#x60;grf&#x60;. | 
**ArtifactPath** | **string** | Server-side stored path; not a client-fetchable URL. | 
**ArtifactUrl** | **string** |  | 
**ThumbnailPath** | **string** |  | 
**ThumbnailUrl** | **string** |  | 
**DurationSeconds** | **int** |  | 
**SizeBytes** | **long** |  | 
**StartedAt** | **DateTime** |  | 
**EndedAt** | **DateTime** |  | [optional] 
**CreatedAt** | **DateTime** |  | 
**UpdatedAt** | **DateTime** |  | 
**User** | **Object** | The owning user, present when the server eager-loads it. Optional - clients must not depend on this being present.  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

