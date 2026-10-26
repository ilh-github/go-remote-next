# OwnerAuditLog

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **int64** |  | 
**ActorUserId** | Pointer to **int64** | The account that reported the transfer. Always the caller. | [optional] 
**Action** | **string** | One of &#x60;file.transfer.upload&#x60;, &#x60;file.transfer.download&#x60;, &#x60;clipboard.sync.upload&#x60;, &#x60;clipboard.sync.download&#x60;.  | 
**ResourceType** | **string** | &#x60;file_transfer&#x60; or &#x60;clipboard_sync&#x60;.  | 
**ResourceId** | **string** | The transfer id the reporter supplied, which is the idempotency key. | 
**Details** | Pointer to **string** | JSON-encoded metadata: byte counts, direction, status and, for file transfers, the filename. Never the transferred content itself.  | [optional] 
**Ip** | Pointer to **string** | The source address the report arrived from. | [optional] 
**CreatedAt** | **time.Time** |  | 

## Methods

### NewOwnerAuditLog

`func NewOwnerAuditLog(id int64, action string, resourceType string, resourceId string, createdAt time.Time, ) *OwnerAuditLog`

NewOwnerAuditLog instantiates a new OwnerAuditLog object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewOwnerAuditLogWithDefaults

`func NewOwnerAuditLogWithDefaults() *OwnerAuditLog`

NewOwnerAuditLogWithDefaults instantiates a new OwnerAuditLog object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *OwnerAuditLog) GetId() int64`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *OwnerAuditLog) GetIdOk() (*int64, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *OwnerAuditLog) SetId(v int64)`

SetId sets Id field to given value.


### GetActorUserId

`func (o *OwnerAuditLog) GetActorUserId() int64`

GetActorUserId returns the ActorUserId field if non-nil, zero value otherwise.

### GetActorUserIdOk

`func (o *OwnerAuditLog) GetActorUserIdOk() (*int64, bool)`

GetActorUserIdOk returns a tuple with the ActorUserId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetActorUserId

`func (o *OwnerAuditLog) SetActorUserId(v int64)`

SetActorUserId sets ActorUserId field to given value.

### HasActorUserId

`func (o *OwnerAuditLog) HasActorUserId() bool`

HasActorUserId returns a boolean if a field has been set.

### GetAction

`func (o *OwnerAuditLog) GetAction() string`

GetAction returns the Action field if non-nil, zero value otherwise.

### GetActionOk

`func (o *OwnerAuditLog) GetActionOk() (*string, bool)`

GetActionOk returns a tuple with the Action field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAction

`func (o *OwnerAuditLog) SetAction(v string)`

SetAction sets Action field to given value.


### GetResourceType

`func (o *OwnerAuditLog) GetResourceType() string`

GetResourceType returns the ResourceType field if non-nil, zero value otherwise.

### GetResourceTypeOk

`func (o *OwnerAuditLog) GetResourceTypeOk() (*string, bool)`

GetResourceTypeOk returns a tuple with the ResourceType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResourceType

`func (o *OwnerAuditLog) SetResourceType(v string)`

SetResourceType sets ResourceType field to given value.


### GetResourceId

`func (o *OwnerAuditLog) GetResourceId() string`

GetResourceId returns the ResourceId field if non-nil, zero value otherwise.

### GetResourceIdOk

`func (o *OwnerAuditLog) GetResourceIdOk() (*string, bool)`

GetResourceIdOk returns a tuple with the ResourceId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResourceId

`func (o *OwnerAuditLog) SetResourceId(v string)`

SetResourceId sets ResourceId field to given value.


### GetDetails

`func (o *OwnerAuditLog) GetDetails() string`

GetDetails returns the Details field if non-nil, zero value otherwise.

### GetDetailsOk

`func (o *OwnerAuditLog) GetDetailsOk() (*string, bool)`

GetDetailsOk returns a tuple with the Details field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDetails

`func (o *OwnerAuditLog) SetDetails(v string)`

SetDetails sets Details field to given value.

### HasDetails

`func (o *OwnerAuditLog) HasDetails() bool`

HasDetails returns a boolean if a field has been set.

### GetIp

`func (o *OwnerAuditLog) GetIp() string`

GetIp returns the Ip field if non-nil, zero value otherwise.

### GetIpOk

`func (o *OwnerAuditLog) GetIpOk() (*string, bool)`

GetIpOk returns a tuple with the Ip field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIp

`func (o *OwnerAuditLog) SetIp(v string)`

SetIp sets Ip field to given value.

### HasIp

`func (o *OwnerAuditLog) HasIp() bool`

HasIp returns a boolean if a field has been set.

### GetCreatedAt

`func (o *OwnerAuditLog) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *OwnerAuditLog) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *OwnerAuditLog) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


