# ArtifactUploadState

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**RecordingId** | **int64** |  | 
**UploadOffset** | **int64** | Bytes staged so far; the &#x60;Upload-Offset&#x60; response header carries the same number. | 
**UploadLength** | **NullableInt64** | The total the client declared, or null when it declared none. | 
**Ext** | **string** | The extension the upload is being staged as. | 

## Methods

### NewArtifactUploadState

`func NewArtifactUploadState(recordingId int64, uploadOffset int64, uploadLength NullableInt64, ext string, ) *ArtifactUploadState`

NewArtifactUploadState instantiates a new ArtifactUploadState object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewArtifactUploadStateWithDefaults

`func NewArtifactUploadStateWithDefaults() *ArtifactUploadState`

NewArtifactUploadStateWithDefaults instantiates a new ArtifactUploadState object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetRecordingId

`func (o *ArtifactUploadState) GetRecordingId() int64`

GetRecordingId returns the RecordingId field if non-nil, zero value otherwise.

### GetRecordingIdOk

`func (o *ArtifactUploadState) GetRecordingIdOk() (*int64, bool)`

GetRecordingIdOk returns a tuple with the RecordingId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRecordingId

`func (o *ArtifactUploadState) SetRecordingId(v int64)`

SetRecordingId sets RecordingId field to given value.


### GetUploadOffset

`func (o *ArtifactUploadState) GetUploadOffset() int64`

GetUploadOffset returns the UploadOffset field if non-nil, zero value otherwise.

### GetUploadOffsetOk

`func (o *ArtifactUploadState) GetUploadOffsetOk() (*int64, bool)`

GetUploadOffsetOk returns a tuple with the UploadOffset field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUploadOffset

`func (o *ArtifactUploadState) SetUploadOffset(v int64)`

SetUploadOffset sets UploadOffset field to given value.


### GetUploadLength

`func (o *ArtifactUploadState) GetUploadLength() int64`

GetUploadLength returns the UploadLength field if non-nil, zero value otherwise.

### GetUploadLengthOk

`func (o *ArtifactUploadState) GetUploadLengthOk() (*int64, bool)`

GetUploadLengthOk returns a tuple with the UploadLength field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUploadLength

`func (o *ArtifactUploadState) SetUploadLength(v int64)`

SetUploadLength sets UploadLength field to given value.


### SetUploadLengthNil

`func (o *ArtifactUploadState) SetUploadLengthNil(b bool)`

 SetUploadLengthNil sets the value for UploadLength to be an explicit nil

### UnsetUploadLength
`func (o *ArtifactUploadState) UnsetUploadLength()`

UnsetUploadLength ensures that no value is present for UploadLength, not even an explicit nil
### GetExt

`func (o *ArtifactUploadState) GetExt() string`

GetExt returns the Ext field if non-nil, zero value otherwise.

### GetExtOk

`func (o *ArtifactUploadState) GetExtOk() (*string, bool)`

GetExtOk returns a tuple with the Ext field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExt

`func (o *ArtifactUploadState) SetExt(v string)`

SetExt sets Ext field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


