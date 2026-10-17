# UserDeviceGroup

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **int64** |  | 
**UserId** | **int64** |  | 
**Name** | **string** |  | 
**Description** | **string** |  | 
**Color** | **string** |  | 
**DeviceIds** | **[]string** |  | 
**DeviceCount** | **int32** |  | 
**CreatedAt** | **time.Time** |  | 
**UpdatedAt** | **time.Time** |  | 

## Methods

### NewUserDeviceGroup

`func NewUserDeviceGroup(id int64, userId int64, name string, description string, color string, deviceIds []string, deviceCount int32, createdAt time.Time, updatedAt time.Time, ) *UserDeviceGroup`

NewUserDeviceGroup instantiates a new UserDeviceGroup object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewUserDeviceGroupWithDefaults

`func NewUserDeviceGroupWithDefaults() *UserDeviceGroup`

NewUserDeviceGroupWithDefaults instantiates a new UserDeviceGroup object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *UserDeviceGroup) GetId() int64`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *UserDeviceGroup) GetIdOk() (*int64, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *UserDeviceGroup) SetId(v int64)`

SetId sets Id field to given value.


### GetUserId

`func (o *UserDeviceGroup) GetUserId() int64`

GetUserId returns the UserId field if non-nil, zero value otherwise.

### GetUserIdOk

`func (o *UserDeviceGroup) GetUserIdOk() (*int64, bool)`

GetUserIdOk returns a tuple with the UserId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUserId

`func (o *UserDeviceGroup) SetUserId(v int64)`

SetUserId sets UserId field to given value.


### GetName

`func (o *UserDeviceGroup) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *UserDeviceGroup) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *UserDeviceGroup) SetName(v string)`

SetName sets Name field to given value.


### GetDescription

`func (o *UserDeviceGroup) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *UserDeviceGroup) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *UserDeviceGroup) SetDescription(v string)`

SetDescription sets Description field to given value.


### GetColor

`func (o *UserDeviceGroup) GetColor() string`

GetColor returns the Color field if non-nil, zero value otherwise.

### GetColorOk

`func (o *UserDeviceGroup) GetColorOk() (*string, bool)`

GetColorOk returns a tuple with the Color field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetColor

`func (o *UserDeviceGroup) SetColor(v string)`

SetColor sets Color field to given value.


### GetDeviceIds

`func (o *UserDeviceGroup) GetDeviceIds() []string`

GetDeviceIds returns the DeviceIds field if non-nil, zero value otherwise.

### GetDeviceIdsOk

`func (o *UserDeviceGroup) GetDeviceIdsOk() (*[]string, bool)`

GetDeviceIdsOk returns a tuple with the DeviceIds field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeviceIds

`func (o *UserDeviceGroup) SetDeviceIds(v []string)`

SetDeviceIds sets DeviceIds field to given value.


### GetDeviceCount

`func (o *UserDeviceGroup) GetDeviceCount() int32`

GetDeviceCount returns the DeviceCount field if non-nil, zero value otherwise.

### GetDeviceCountOk

`func (o *UserDeviceGroup) GetDeviceCountOk() (*int32, bool)`

GetDeviceCountOk returns a tuple with the DeviceCount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeviceCount

`func (o *UserDeviceGroup) SetDeviceCount(v int32)`

SetDeviceCount sets DeviceCount field to given value.


### GetCreatedAt

`func (o *UserDeviceGroup) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *UserDeviceGroup) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *UserDeviceGroup) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.


### GetUpdatedAt

`func (o *UserDeviceGroup) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *UserDeviceGroup) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *UserDeviceGroup) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


