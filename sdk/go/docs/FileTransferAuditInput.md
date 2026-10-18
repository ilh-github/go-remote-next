# FileTransferAuditInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DeviceId** | **string** | A device owned by the authenticated caller. | 
**TransferId** | **string** | Stable id used to make retries idempotent. | 
**Direction** | **string** | Upload is controller to host; download is host to controller. | 
**Status** | **string** |  | 
**Filename** | Pointer to **string** |  | [optional] 
**BytesTransferred** | Pointer to **int64** |  | [optional] 
**TotalBytes** | Pointer to **int64** | Zero means the controller did not know the total size. | [optional] 
**Error** | Pointer to **string** | Human-readable failure detail when status is failed. | [optional] 

## Methods

### NewFileTransferAuditInput

`func NewFileTransferAuditInput(deviceId string, transferId string, direction string, status string, ) *FileTransferAuditInput`

NewFileTransferAuditInput instantiates a new FileTransferAuditInput object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewFileTransferAuditInputWithDefaults

`func NewFileTransferAuditInputWithDefaults() *FileTransferAuditInput`

NewFileTransferAuditInputWithDefaults instantiates a new FileTransferAuditInput object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDeviceId

`func (o *FileTransferAuditInput) GetDeviceId() string`

GetDeviceId returns the DeviceId field if non-nil, zero value otherwise.

### GetDeviceIdOk

`func (o *FileTransferAuditInput) GetDeviceIdOk() (*string, bool)`

GetDeviceIdOk returns a tuple with the DeviceId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeviceId

`func (o *FileTransferAuditInput) SetDeviceId(v string)`

SetDeviceId sets DeviceId field to given value.


### GetTransferId

`func (o *FileTransferAuditInput) GetTransferId() string`

GetTransferId returns the TransferId field if non-nil, zero value otherwise.

### GetTransferIdOk

`func (o *FileTransferAuditInput) GetTransferIdOk() (*string, bool)`

GetTransferIdOk returns a tuple with the TransferId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTransferId

`func (o *FileTransferAuditInput) SetTransferId(v string)`

SetTransferId sets TransferId field to given value.


### GetDirection

`func (o *FileTransferAuditInput) GetDirection() string`

GetDirection returns the Direction field if non-nil, zero value otherwise.

### GetDirectionOk

`func (o *FileTransferAuditInput) GetDirectionOk() (*string, bool)`

GetDirectionOk returns a tuple with the Direction field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDirection

`func (o *FileTransferAuditInput) SetDirection(v string)`

SetDirection sets Direction field to given value.


### GetStatus

`func (o *FileTransferAuditInput) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *FileTransferAuditInput) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *FileTransferAuditInput) SetStatus(v string)`

SetStatus sets Status field to given value.


### GetFilename

`func (o *FileTransferAuditInput) GetFilename() string`

GetFilename returns the Filename field if non-nil, zero value otherwise.

### GetFilenameOk

`func (o *FileTransferAuditInput) GetFilenameOk() (*string, bool)`

GetFilenameOk returns a tuple with the Filename field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFilename

`func (o *FileTransferAuditInput) SetFilename(v string)`

SetFilename sets Filename field to given value.

### HasFilename

`func (o *FileTransferAuditInput) HasFilename() bool`

HasFilename returns a boolean if a field has been set.

### GetBytesTransferred

`func (o *FileTransferAuditInput) GetBytesTransferred() int64`

GetBytesTransferred returns the BytesTransferred field if non-nil, zero value otherwise.

### GetBytesTransferredOk

`func (o *FileTransferAuditInput) GetBytesTransferredOk() (*int64, bool)`

GetBytesTransferredOk returns a tuple with the BytesTransferred field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBytesTransferred

`func (o *FileTransferAuditInput) SetBytesTransferred(v int64)`

SetBytesTransferred sets BytesTransferred field to given value.

### HasBytesTransferred

`func (o *FileTransferAuditInput) HasBytesTransferred() bool`

HasBytesTransferred returns a boolean if a field has been set.

### GetTotalBytes

`func (o *FileTransferAuditInput) GetTotalBytes() int64`

GetTotalBytes returns the TotalBytes field if non-nil, zero value otherwise.

### GetTotalBytesOk

`func (o *FileTransferAuditInput) GetTotalBytesOk() (*int64, bool)`

GetTotalBytesOk returns a tuple with the TotalBytes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTotalBytes

`func (o *FileTransferAuditInput) SetTotalBytes(v int64)`

SetTotalBytes sets TotalBytes field to given value.

### HasTotalBytes

`func (o *FileTransferAuditInput) HasTotalBytes() bool`

HasTotalBytes returns a boolean if a field has been set.

### GetError

`func (o *FileTransferAuditInput) GetError() string`

GetError returns the Error field if non-nil, zero value otherwise.

### GetErrorOk

`func (o *FileTransferAuditInput) GetErrorOk() (*string, bool)`

GetErrorOk returns a tuple with the Error field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetError

`func (o *FileTransferAuditInput) SetError(v string)`

SetError sets Error field to given value.

### HasError

`func (o *FileTransferAuditInput) HasError() bool`

HasError returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


