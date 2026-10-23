# ClipboardAuditInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DeviceId** | **string** | A device owned by the authenticated caller. | 
**TransferId** | **string** | Stable id used to make retries idempotent. | 
**Direction** | **string** | Which way the clipboard went: upload is controller to host, download is host to controller. | 
**Status** | **string** |  | 
**MimeType** | **string** | The clipboard format, e.g. &#x60;text/plain&#x60;. Metadata only. | 
**Bytes** | Pointer to **int64** | Size of the payload that crossed the data channel. | [optional] 

## Methods

### NewClipboardAuditInput

`func NewClipboardAuditInput(deviceId string, transferId string, direction string, status string, mimeType string, ) *ClipboardAuditInput`

NewClipboardAuditInput instantiates a new ClipboardAuditInput object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewClipboardAuditInputWithDefaults

`func NewClipboardAuditInputWithDefaults() *ClipboardAuditInput`

NewClipboardAuditInputWithDefaults instantiates a new ClipboardAuditInput object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDeviceId

`func (o *ClipboardAuditInput) GetDeviceId() string`

GetDeviceId returns the DeviceId field if non-nil, zero value otherwise.

### GetDeviceIdOk

`func (o *ClipboardAuditInput) GetDeviceIdOk() (*string, bool)`

GetDeviceIdOk returns a tuple with the DeviceId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeviceId

`func (o *ClipboardAuditInput) SetDeviceId(v string)`

SetDeviceId sets DeviceId field to given value.


### GetTransferId

`func (o *ClipboardAuditInput) GetTransferId() string`

GetTransferId returns the TransferId field if non-nil, zero value otherwise.

### GetTransferIdOk

`func (o *ClipboardAuditInput) GetTransferIdOk() (*string, bool)`

GetTransferIdOk returns a tuple with the TransferId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTransferId

`func (o *ClipboardAuditInput) SetTransferId(v string)`

SetTransferId sets TransferId field to given value.


### GetDirection

`func (o *ClipboardAuditInput) GetDirection() string`

GetDirection returns the Direction field if non-nil, zero value otherwise.

### GetDirectionOk

`func (o *ClipboardAuditInput) GetDirectionOk() (*string, bool)`

GetDirectionOk returns a tuple with the Direction field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDirection

`func (o *ClipboardAuditInput) SetDirection(v string)`

SetDirection sets Direction field to given value.


### GetStatus

`func (o *ClipboardAuditInput) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *ClipboardAuditInput) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *ClipboardAuditInput) SetStatus(v string)`

SetStatus sets Status field to given value.


### GetMimeType

`func (o *ClipboardAuditInput) GetMimeType() string`

GetMimeType returns the MimeType field if non-nil, zero value otherwise.

### GetMimeTypeOk

`func (o *ClipboardAuditInput) GetMimeTypeOk() (*string, bool)`

GetMimeTypeOk returns a tuple with the MimeType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMimeType

`func (o *ClipboardAuditInput) SetMimeType(v string)`

SetMimeType sets MimeType field to given value.


### GetBytes

`func (o *ClipboardAuditInput) GetBytes() int64`

GetBytes returns the Bytes field if non-nil, zero value otherwise.

### GetBytesOk

`func (o *ClipboardAuditInput) GetBytesOk() (*int64, bool)`

GetBytesOk returns a tuple with the Bytes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBytes

`func (o *ClipboardAuditInput) SetBytes(v int64)`

SetBytes sets Bytes field to given value.

### HasBytes

`func (o *ClipboardAuditInput) HasBytes() bool`

HasBytes returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


