

# ArtifactUploadState

How much of a resumable recording upload the server currently holds.

## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**recordingId** | **Long** |  |  |
|**uploadOffset** | **Long** | Bytes staged so far; the &#x60;Upload-Offset&#x60; response header carries the same number. |  |
|**uploadLength** | **Long** | The total the client declared, or null when it declared none. |  |
|**ext** | **String** | The extension the upload is being staged as. |  |



