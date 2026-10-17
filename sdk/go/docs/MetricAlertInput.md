# MetricAlertInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Enabled** | **bool** |  | 
**CpuPercent** | **int32** | 0 disables the CPU threshold. | 
**MemoryPercent** | **int32** | 0 disables the memory threshold. | 
**DiskPercent** | **int32** | 0 disables the disk threshold. | 
**SustainedMinutes** | **int32** | 0 alerts on the first over-threshold sample. | 
**RepeatMinutes** | **int32** | 0 sends one alert per continuous breach. | 
**SnoozedUntil** | Pointer to **NullableTime** |  | [optional] 

## Methods

### NewMetricAlertInput

`func NewMetricAlertInput(enabled bool, cpuPercent int32, memoryPercent int32, diskPercent int32, sustainedMinutes int32, repeatMinutes int32, ) *MetricAlertInput`

NewMetricAlertInput instantiates a new MetricAlertInput object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMetricAlertInputWithDefaults

`func NewMetricAlertInputWithDefaults() *MetricAlertInput`

NewMetricAlertInputWithDefaults instantiates a new MetricAlertInput object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetEnabled

`func (o *MetricAlertInput) GetEnabled() bool`

GetEnabled returns the Enabled field if non-nil, zero value otherwise.

### GetEnabledOk

`func (o *MetricAlertInput) GetEnabledOk() (*bool, bool)`

GetEnabledOk returns a tuple with the Enabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnabled

`func (o *MetricAlertInput) SetEnabled(v bool)`

SetEnabled sets Enabled field to given value.


### GetCpuPercent

`func (o *MetricAlertInput) GetCpuPercent() int32`

GetCpuPercent returns the CpuPercent field if non-nil, zero value otherwise.

### GetCpuPercentOk

`func (o *MetricAlertInput) GetCpuPercentOk() (*int32, bool)`

GetCpuPercentOk returns a tuple with the CpuPercent field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCpuPercent

`func (o *MetricAlertInput) SetCpuPercent(v int32)`

SetCpuPercent sets CpuPercent field to given value.


### GetMemoryPercent

`func (o *MetricAlertInput) GetMemoryPercent() int32`

GetMemoryPercent returns the MemoryPercent field if non-nil, zero value otherwise.

### GetMemoryPercentOk

`func (o *MetricAlertInput) GetMemoryPercentOk() (*int32, bool)`

GetMemoryPercentOk returns a tuple with the MemoryPercent field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMemoryPercent

`func (o *MetricAlertInput) SetMemoryPercent(v int32)`

SetMemoryPercent sets MemoryPercent field to given value.


### GetDiskPercent

`func (o *MetricAlertInput) GetDiskPercent() int32`

GetDiskPercent returns the DiskPercent field if non-nil, zero value otherwise.

### GetDiskPercentOk

`func (o *MetricAlertInput) GetDiskPercentOk() (*int32, bool)`

GetDiskPercentOk returns a tuple with the DiskPercent field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDiskPercent

`func (o *MetricAlertInput) SetDiskPercent(v int32)`

SetDiskPercent sets DiskPercent field to given value.


### GetSustainedMinutes

`func (o *MetricAlertInput) GetSustainedMinutes() int32`

GetSustainedMinutes returns the SustainedMinutes field if non-nil, zero value otherwise.

### GetSustainedMinutesOk

`func (o *MetricAlertInput) GetSustainedMinutesOk() (*int32, bool)`

GetSustainedMinutesOk returns a tuple with the SustainedMinutes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSustainedMinutes

`func (o *MetricAlertInput) SetSustainedMinutes(v int32)`

SetSustainedMinutes sets SustainedMinutes field to given value.


### GetRepeatMinutes

`func (o *MetricAlertInput) GetRepeatMinutes() int32`

GetRepeatMinutes returns the RepeatMinutes field if non-nil, zero value otherwise.

### GetRepeatMinutesOk

`func (o *MetricAlertInput) GetRepeatMinutesOk() (*int32, bool)`

GetRepeatMinutesOk returns a tuple with the RepeatMinutes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepeatMinutes

`func (o *MetricAlertInput) SetRepeatMinutes(v int32)`

SetRepeatMinutes sets RepeatMinutes field to given value.


### GetSnoozedUntil

`func (o *MetricAlertInput) GetSnoozedUntil() time.Time`

GetSnoozedUntil returns the SnoozedUntil field if non-nil, zero value otherwise.

### GetSnoozedUntilOk

`func (o *MetricAlertInput) GetSnoozedUntilOk() (*time.Time, bool)`

GetSnoozedUntilOk returns a tuple with the SnoozedUntil field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSnoozedUntil

`func (o *MetricAlertInput) SetSnoozedUntil(v time.Time)`

SetSnoozedUntil sets SnoozedUntil field to given value.

### HasSnoozedUntil

`func (o *MetricAlertInput) HasSnoozedUntil() bool`

HasSnoozedUntil returns a boolean if a field has been set.

### SetSnoozedUntilNil

`func (o *MetricAlertInput) SetSnoozedUntilNil(b bool)`

 SetSnoozedUntilNil sets the value for SnoozedUntil to be an explicit nil

### UnsetSnoozedUntil
`func (o *MetricAlertInput) UnsetSnoozedUntil()`

UnsetSnoozedUntil ensures that no value is present for SnoozedUntil, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


