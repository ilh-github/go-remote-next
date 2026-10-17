# MetricAlertRule

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DeviceId** | **string** |  | 
**Enabled** | **bool** |  | 
**CpuPercent** | **int32** |  | 
**MemoryPercent** | **int32** |  | 
**DiskPercent** | **int32** |  | 
**SustainedMinutes** | **int32** |  | 
**RepeatMinutes** | **int32** |  | 
**SnoozedUntil** | Pointer to **NullableTime** |  | [optional] 
**Current** | Pointer to [**NullableMetricSnapshot**](MetricSnapshot.md) |  | [optional] 
**States** | [**MetricAlertStates**](MetricAlertStates.md) |  | 
**UpdatedAt** | Pointer to **NullableTime** |  | [optional] 

## Methods

### NewMetricAlertRule

`func NewMetricAlertRule(deviceId string, enabled bool, cpuPercent int32, memoryPercent int32, diskPercent int32, sustainedMinutes int32, repeatMinutes int32, states MetricAlertStates, ) *MetricAlertRule`

NewMetricAlertRule instantiates a new MetricAlertRule object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMetricAlertRuleWithDefaults

`func NewMetricAlertRuleWithDefaults() *MetricAlertRule`

NewMetricAlertRuleWithDefaults instantiates a new MetricAlertRule object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDeviceId

`func (o *MetricAlertRule) GetDeviceId() string`

GetDeviceId returns the DeviceId field if non-nil, zero value otherwise.

### GetDeviceIdOk

`func (o *MetricAlertRule) GetDeviceIdOk() (*string, bool)`

GetDeviceIdOk returns a tuple with the DeviceId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeviceId

`func (o *MetricAlertRule) SetDeviceId(v string)`

SetDeviceId sets DeviceId field to given value.


### GetEnabled

`func (o *MetricAlertRule) GetEnabled() bool`

GetEnabled returns the Enabled field if non-nil, zero value otherwise.

### GetEnabledOk

`func (o *MetricAlertRule) GetEnabledOk() (*bool, bool)`

GetEnabledOk returns a tuple with the Enabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnabled

`func (o *MetricAlertRule) SetEnabled(v bool)`

SetEnabled sets Enabled field to given value.


### GetCpuPercent

`func (o *MetricAlertRule) GetCpuPercent() int32`

GetCpuPercent returns the CpuPercent field if non-nil, zero value otherwise.

### GetCpuPercentOk

`func (o *MetricAlertRule) GetCpuPercentOk() (*int32, bool)`

GetCpuPercentOk returns a tuple with the CpuPercent field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCpuPercent

`func (o *MetricAlertRule) SetCpuPercent(v int32)`

SetCpuPercent sets CpuPercent field to given value.


### GetMemoryPercent

`func (o *MetricAlertRule) GetMemoryPercent() int32`

GetMemoryPercent returns the MemoryPercent field if non-nil, zero value otherwise.

### GetMemoryPercentOk

`func (o *MetricAlertRule) GetMemoryPercentOk() (*int32, bool)`

GetMemoryPercentOk returns a tuple with the MemoryPercent field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMemoryPercent

`func (o *MetricAlertRule) SetMemoryPercent(v int32)`

SetMemoryPercent sets MemoryPercent field to given value.


### GetDiskPercent

`func (o *MetricAlertRule) GetDiskPercent() int32`

GetDiskPercent returns the DiskPercent field if non-nil, zero value otherwise.

### GetDiskPercentOk

`func (o *MetricAlertRule) GetDiskPercentOk() (*int32, bool)`

GetDiskPercentOk returns a tuple with the DiskPercent field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDiskPercent

`func (o *MetricAlertRule) SetDiskPercent(v int32)`

SetDiskPercent sets DiskPercent field to given value.


### GetSustainedMinutes

`func (o *MetricAlertRule) GetSustainedMinutes() int32`

GetSustainedMinutes returns the SustainedMinutes field if non-nil, zero value otherwise.

### GetSustainedMinutesOk

`func (o *MetricAlertRule) GetSustainedMinutesOk() (*int32, bool)`

GetSustainedMinutesOk returns a tuple with the SustainedMinutes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSustainedMinutes

`func (o *MetricAlertRule) SetSustainedMinutes(v int32)`

SetSustainedMinutes sets SustainedMinutes field to given value.


### GetRepeatMinutes

`func (o *MetricAlertRule) GetRepeatMinutes() int32`

GetRepeatMinutes returns the RepeatMinutes field if non-nil, zero value otherwise.

### GetRepeatMinutesOk

`func (o *MetricAlertRule) GetRepeatMinutesOk() (*int32, bool)`

GetRepeatMinutesOk returns a tuple with the RepeatMinutes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepeatMinutes

`func (o *MetricAlertRule) SetRepeatMinutes(v int32)`

SetRepeatMinutes sets RepeatMinutes field to given value.


### GetSnoozedUntil

`func (o *MetricAlertRule) GetSnoozedUntil() time.Time`

GetSnoozedUntil returns the SnoozedUntil field if non-nil, zero value otherwise.

### GetSnoozedUntilOk

`func (o *MetricAlertRule) GetSnoozedUntilOk() (*time.Time, bool)`

GetSnoozedUntilOk returns a tuple with the SnoozedUntil field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSnoozedUntil

`func (o *MetricAlertRule) SetSnoozedUntil(v time.Time)`

SetSnoozedUntil sets SnoozedUntil field to given value.

### HasSnoozedUntil

`func (o *MetricAlertRule) HasSnoozedUntil() bool`

HasSnoozedUntil returns a boolean if a field has been set.

### SetSnoozedUntilNil

`func (o *MetricAlertRule) SetSnoozedUntilNil(b bool)`

 SetSnoozedUntilNil sets the value for SnoozedUntil to be an explicit nil

### UnsetSnoozedUntil
`func (o *MetricAlertRule) UnsetSnoozedUntil()`

UnsetSnoozedUntil ensures that no value is present for SnoozedUntil, not even an explicit nil
### GetCurrent

`func (o *MetricAlertRule) GetCurrent() MetricSnapshot`

GetCurrent returns the Current field if non-nil, zero value otherwise.

### GetCurrentOk

`func (o *MetricAlertRule) GetCurrentOk() (*MetricSnapshot, bool)`

GetCurrentOk returns a tuple with the Current field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCurrent

`func (o *MetricAlertRule) SetCurrent(v MetricSnapshot)`

SetCurrent sets Current field to given value.

### HasCurrent

`func (o *MetricAlertRule) HasCurrent() bool`

HasCurrent returns a boolean if a field has been set.

### SetCurrentNil

`func (o *MetricAlertRule) SetCurrentNil(b bool)`

 SetCurrentNil sets the value for Current to be an explicit nil

### UnsetCurrent
`func (o *MetricAlertRule) UnsetCurrent()`

UnsetCurrent ensures that no value is present for Current, not even an explicit nil
### GetStates

`func (o *MetricAlertRule) GetStates() MetricAlertStates`

GetStates returns the States field if non-nil, zero value otherwise.

### GetStatesOk

`func (o *MetricAlertRule) GetStatesOk() (*MetricAlertStates, bool)`

GetStatesOk returns a tuple with the States field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStates

`func (o *MetricAlertRule) SetStates(v MetricAlertStates)`

SetStates sets States field to given value.


### GetUpdatedAt

`func (o *MetricAlertRule) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *MetricAlertRule) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *MetricAlertRule) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *MetricAlertRule) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### SetUpdatedAtNil

`func (o *MetricAlertRule) SetUpdatedAtNil(b bool)`

 SetUpdatedAtNil sets the value for UpdatedAt to be an explicit nil

### UnsetUpdatedAt
`func (o *MetricAlertRule) UnsetUpdatedAt()`

UnsetUpdatedAt ensures that no value is present for UpdatedAt, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


