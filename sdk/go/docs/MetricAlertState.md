# MetricAlertState

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ExceededSince** | Pointer to **NullableTime** |  | [optional] 
**LastAlertedAt** | Pointer to **NullableTime** |  | [optional] 

## Methods

### NewMetricAlertState

`func NewMetricAlertState() *MetricAlertState`

NewMetricAlertState instantiates a new MetricAlertState object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMetricAlertStateWithDefaults

`func NewMetricAlertStateWithDefaults() *MetricAlertState`

NewMetricAlertStateWithDefaults instantiates a new MetricAlertState object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetExceededSince

`func (o *MetricAlertState) GetExceededSince() time.Time`

GetExceededSince returns the ExceededSince field if non-nil, zero value otherwise.

### GetExceededSinceOk

`func (o *MetricAlertState) GetExceededSinceOk() (*time.Time, bool)`

GetExceededSinceOk returns a tuple with the ExceededSince field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExceededSince

`func (o *MetricAlertState) SetExceededSince(v time.Time)`

SetExceededSince sets ExceededSince field to given value.

### HasExceededSince

`func (o *MetricAlertState) HasExceededSince() bool`

HasExceededSince returns a boolean if a field has been set.

### SetExceededSinceNil

`func (o *MetricAlertState) SetExceededSinceNil(b bool)`

 SetExceededSinceNil sets the value for ExceededSince to be an explicit nil

### UnsetExceededSince
`func (o *MetricAlertState) UnsetExceededSince()`

UnsetExceededSince ensures that no value is present for ExceededSince, not even an explicit nil
### GetLastAlertedAt

`func (o *MetricAlertState) GetLastAlertedAt() time.Time`

GetLastAlertedAt returns the LastAlertedAt field if non-nil, zero value otherwise.

### GetLastAlertedAtOk

`func (o *MetricAlertState) GetLastAlertedAtOk() (*time.Time, bool)`

GetLastAlertedAtOk returns a tuple with the LastAlertedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastAlertedAt

`func (o *MetricAlertState) SetLastAlertedAt(v time.Time)`

SetLastAlertedAt sets LastAlertedAt field to given value.

### HasLastAlertedAt

`func (o *MetricAlertState) HasLastAlertedAt() bool`

HasLastAlertedAt returns a boolean if a field has been set.

### SetLastAlertedAtNil

`func (o *MetricAlertState) SetLastAlertedAtNil(b bool)`

 SetLastAlertedAtNil sets the value for LastAlertedAt to be an explicit nil

### UnsetLastAlertedAt
`func (o *MetricAlertState) UnsetLastAlertedAt()`

UnsetLastAlertedAt ensures that no value is present for LastAlertedAt, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


