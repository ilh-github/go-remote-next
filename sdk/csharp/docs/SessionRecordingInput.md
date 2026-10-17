# QuickDesk.Signaling.Client.Model.SessionRecordingInput
The index row a producer creates before (or instead of) uploading media. `device_id` is recorded as a label; it is not checked against the caller's devices, because a headless producer may be indexing a capture for a machine it is not bound to. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DeviceId** | **string** | The device the capture came from. | 
**DeviceName** | **string** |  | [optional] 
**SessionId** | **string** | The control session this recording belongs to, when there is one. | [optional] 
**Status** | **string** | How far the capture got when this row was written. | 
**Format** | **string** | Free-form label; the upload route overwrites it from the file extension. | [optional] 
**ArtifactPath** | **string** | Server-side path. Set by the upload route; clients do not need to send it. | [optional] 
**ArtifactUrl** | **string** |  | [optional] 
**ThumbnailPath** | **string** |  | [optional] 
**DurationSeconds** | **int** |  | [optional] 
**SizeBytes** | **int** |  | [optional] 
**StartedAt** | **DateTime?** | Defaults to the time the row was created when omitted. | [optional] 
**EndedAt** | **DateTime?** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

