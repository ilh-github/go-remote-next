# OwnerAuditLogPage

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Items** | [**[]OwnerAuditLog**](OwnerAuditLog.md) |  | 
**NextCursor** | Pointer to **string** |  | [optional] 
**Total** | Pointer to **int64** |  | [optional] 

## Methods

### NewOwnerAuditLogPage

`func NewOwnerAuditLogPage(items []OwnerAuditLog, ) *OwnerAuditLogPage`

NewOwnerAuditLogPage instantiates a new OwnerAuditLogPage object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewOwnerAuditLogPageWithDefaults

`func NewOwnerAuditLogPageWithDefaults() *OwnerAuditLogPage`

NewOwnerAuditLogPageWithDefaults instantiates a new OwnerAuditLogPage object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetItems

`func (o *OwnerAuditLogPage) GetItems() []OwnerAuditLog`

GetItems returns the Items field if non-nil, zero value otherwise.

### GetItemsOk

`func (o *OwnerAuditLogPage) GetItemsOk() (*[]OwnerAuditLog, bool)`

GetItemsOk returns a tuple with the Items field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetItems

`func (o *OwnerAuditLogPage) SetItems(v []OwnerAuditLog)`

SetItems sets Items field to given value.


### GetNextCursor

`func (o *OwnerAuditLogPage) GetNextCursor() string`

GetNextCursor returns the NextCursor field if non-nil, zero value otherwise.

### GetNextCursorOk

`func (o *OwnerAuditLogPage) GetNextCursorOk() (*string, bool)`

GetNextCursorOk returns a tuple with the NextCursor field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNextCursor

`func (o *OwnerAuditLogPage) SetNextCursor(v string)`

SetNextCursor sets NextCursor field to given value.

### HasNextCursor

`func (o *OwnerAuditLogPage) HasNextCursor() bool`

HasNextCursor returns a boolean if a field has been set.

### GetTotal

`func (o *OwnerAuditLogPage) GetTotal() int64`

GetTotal returns the Total field if non-nil, zero value otherwise.

### GetTotalOk

`func (o *OwnerAuditLogPage) GetTotalOk() (*int64, bool)`

GetTotalOk returns a tuple with the Total field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTotal

`func (o *OwnerAuditLogPage) SetTotal(v int64)`

SetTotal sets Total field to given value.

### HasTotal

`func (o *OwnerAuditLogPage) HasTotal() bool`

HasTotal returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


