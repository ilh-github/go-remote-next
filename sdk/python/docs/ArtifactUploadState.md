# ArtifactUploadState

How much of a resumable recording upload the server currently holds.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**recording_id** | **int** |  | 
**upload_offset** | **int** | Bytes staged so far; the &#x60;Upload-Offset&#x60; response header carries the same number. | 
**upload_length** | **int** | The total the client declared, or null when it declared none. | 
**ext** | **str** | The extension the upload is being staged as. | 

## Example

```python
from quickdesk_signaling.models.artifact_upload_state import ArtifactUploadState

# TODO update the JSON string below
json = "{}"
# create an instance of ArtifactUploadState from a JSON string
artifact_upload_state_instance = ArtifactUploadState.from_json(json)
# print the JSON string representation of the object
print(ArtifactUploadState.to_json())

# convert the object into a dict
artifact_upload_state_dict = artifact_upload_state_instance.to_dict()
# create an instance of ArtifactUploadState from a dict
artifact_upload_state_from_dict = ArtifactUploadState.from_dict(artifact_upload_state_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


