# MetricSnapshot

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CapturedAt** | **time.Time** |  | 
**CpuPercent** | **float32** |  | 
**MemoryPercent** | **float32** |  | 
**DiskPercent** | **float32** |  | 
**DiskMount** | **string** |  | 

## Methods

### NewMetricSnapshot

`func NewMetricSnapshot(capturedAt time.Time, cpuPercent float32, memoryPercent float32, diskPercent float32, diskMount string, ) *MetricSnapshot`

NewMetricSnapshot instantiates a new MetricSnapshot object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMetricSnapshotWithDefaults

`func NewMetricSnapshotWithDefaults() *MetricSnapshot`

NewMetricSnapshotWithDefaults instantiates a new MetricSnapshot object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCapturedAt

`func (o *MetricSnapshot) GetCapturedAt() time.Time`

GetCapturedAt returns the CapturedAt field if non-nil, zero value otherwise.

### GetCapturedAtOk

`func (o *MetricSnapshot) GetCapturedAtOk() (*time.Time, bool)`

GetCapturedAtOk returns a tuple with the CapturedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCapturedAt

`func (o *MetricSnapshot) SetCapturedAt(v time.Time)`

SetCapturedAt sets CapturedAt field to given value.


### GetCpuPercent

`func (o *MetricSnapshot) GetCpuPercent() float32`

GetCpuPercent returns the CpuPercent field if non-nil, zero value otherwise.

### GetCpuPercentOk

`func (o *MetricSnapshot) GetCpuPercentOk() (*float32, bool)`

GetCpuPercentOk returns a tuple with the CpuPercent field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCpuPercent

`func (o *MetricSnapshot) SetCpuPercent(v float32)`

SetCpuPercent sets CpuPercent field to given value.


### GetMemoryPercent

`func (o *MetricSnapshot) GetMemoryPercent() float32`

GetMemoryPercent returns the MemoryPercent field if non-nil, zero value otherwise.

### GetMemoryPercentOk

`func (o *MetricSnapshot) GetMemoryPercentOk() (*float32, bool)`

GetMemoryPercentOk returns a tuple with the MemoryPercent field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMemoryPercent

`func (o *MetricSnapshot) SetMemoryPercent(v float32)`

SetMemoryPercent sets MemoryPercent field to given value.


### GetDiskPercent

`func (o *MetricSnapshot) GetDiskPercent() float32`

GetDiskPercent returns the DiskPercent field if non-nil, zero value otherwise.

### GetDiskPercentOk

`func (o *MetricSnapshot) GetDiskPercentOk() (*float32, bool)`

GetDiskPercentOk returns a tuple with the DiskPercent field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDiskPercent

`func (o *MetricSnapshot) SetDiskPercent(v float32)`

SetDiskPercent sets DiskPercent field to given value.


### GetDiskMount

`func (o *MetricSnapshot) GetDiskMount() string`

GetDiskMount returns the DiskMount field if non-nil, zero value otherwise.

### GetDiskMountOk

`func (o *MetricSnapshot) GetDiskMountOk() (*string, bool)`

GetDiskMountOk returns a tuple with the DiskMount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDiskMount

`func (o *MetricSnapshot) SetDiskMount(v string)`

SetDiskMount sets DiskMount field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


