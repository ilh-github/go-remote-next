# # SessionRecordingInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**device_id** | **string** | The device the capture came from. |
**device_name** | **string** |  | [optional]
**session_id** | **string** | The control session this recording belongs to, when there is one. | [optional]
**status** | **string** | How far the capture got when this row was written. |
**format** | **string** | Free-form label; the upload route overwrites it from the file extension. | [optional]
**artifact_path** | **string** | Server-side path. Set by the upload route; clients do not need to send it. | [optional]
**artifact_url** | **string** |  | [optional]
**thumbnail_path** | **string** |  | [optional]
**duration_seconds** | **int** |  | [optional]
**size_bytes** | **int** |  | [optional]
**started_at** | **\DateTime** | Defaults to the time the row was created when omitted. | [optional]
**ended_at** | **\DateTime** |  | [optional]

[[Back to Model list]](../../README.md#models) [[Back to API list]](../../README.md#endpoints) [[Back to README]](../../README.md)
