# MetricAlertStates

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Cpu** | [**MetricAlertState**](MetricAlertState.md) |  | 
**Memory** | [**MetricAlertState**](MetricAlertState.md) |  | 
**Disk** | [**MetricAlertState**](MetricAlertState.md) |  | 

## Methods

### NewMetricAlertStates

`func NewMetricAlertStates(cpu MetricAlertState, memory MetricAlertState, disk MetricAlertState, ) *MetricAlertStates`

NewMetricAlertStates instantiates a new MetricAlertStates object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMetricAlertStatesWithDefaults

`func NewMetricAlertStatesWithDefaults() *MetricAlertStates`

NewMetricAlertStatesWithDefaults instantiates a new MetricAlertStates object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCpu

`func (o *MetricAlertStates) GetCpu() MetricAlertState`

GetCpu returns the Cpu field if non-nil, zero value otherwise.

### GetCpuOk

`func (o *MetricAlertStates) GetCpuOk() (*MetricAlertState, bool)`

GetCpuOk returns a tuple with the Cpu field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCpu

`func (o *MetricAlertStates) SetCpu(v MetricAlertState)`

SetCpu sets Cpu field to given value.


### GetMemory

`func (o *MetricAlertStates) GetMemory() MetricAlertState`

GetMemory returns the Memory field if non-nil, zero value otherwise.

### GetMemoryOk

`func (o *MetricAlertStates) GetMemoryOk() (*MetricAlertState, bool)`

GetMemoryOk returns a tuple with the Memory field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMemory

`func (o *MetricAlertStates) SetMemory(v MetricAlertState)`

SetMemory sets Memory field to given value.


### GetDisk

`func (o *MetricAlertStates) GetDisk() MetricAlertState`

GetDisk returns the Disk field if non-nil, zero value otherwise.

### GetDiskOk

`func (o *MetricAlertStates) GetDiskOk() (*MetricAlertState, bool)`

GetDiskOk returns a tuple with the Disk field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDisk

`func (o *MetricAlertStates) SetDisk(v MetricAlertState)`

SetDisk sets Disk field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


