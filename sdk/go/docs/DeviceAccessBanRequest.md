# DeviceAccessBanRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Source** | **string** | Exact IPv4/IPv6 address or CIDR block to deny permanently. | 

## Methods

### NewDeviceAccessBanRequest

`func NewDeviceAccessBanRequest(source string, ) *DeviceAccessBanRequest`

NewDeviceAccessBanRequest instantiates a new DeviceAccessBanRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDeviceAccessBanRequestWithDefaults

`func NewDeviceAccessBanRequestWithDefaults() *DeviceAccessBanRequest`

NewDeviceAccessBanRequestWithDefaults instantiates a new DeviceAccessBanRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSource

`func (o *DeviceAccessBanRequest) GetSource() string`

GetSource returns the Source field if non-nil, zero value otherwise.

### GetSourceOk

`func (o *DeviceAccessBanRequest) GetSourceOk() (*string, bool)`

GetSourceOk returns a tuple with the Source field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSource

`func (o *DeviceAccessBanRequest) SetSource(v string)`

SetSource sets Source field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


