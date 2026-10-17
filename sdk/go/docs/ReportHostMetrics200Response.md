# ReportHostMetrics200Response

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DeviceId** | **string** |  | 
**CapturedAt** | **time.Time** |  | 

## Methods

### NewReportHostMetrics200Response

`func NewReportHostMetrics200Response(deviceId string, capturedAt time.Time, ) *ReportHostMetrics200Response`

NewReportHostMetrics200Response instantiates a new ReportHostMetrics200Response object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewReportHostMetrics200ResponseWithDefaults

`func NewReportHostMetrics200ResponseWithDefaults() *ReportHostMetrics200Response`

NewReportHostMetrics200ResponseWithDefaults instantiates a new ReportHostMetrics200Response object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDeviceId

`func (o *ReportHostMetrics200Response) GetDeviceId() string`

GetDeviceId returns the DeviceId field if non-nil, zero value otherwise.

### GetDeviceIdOk

`func (o *ReportHostMetrics200Response) GetDeviceIdOk() (*string, bool)`

GetDeviceIdOk returns a tuple with the DeviceId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeviceId

`func (o *ReportHostMetrics200Response) SetDeviceId(v string)`

SetDeviceId sets DeviceId field to given value.


### GetCapturedAt

`func (o *ReportHostMetrics200Response) GetCapturedAt() time.Time`

GetCapturedAt returns the CapturedAt field if non-nil, zero value otherwise.

### GetCapturedAtOk

`func (o *ReportHostMetrics200Response) GetCapturedAtOk() (*time.Time, bool)`

GetCapturedAtOk returns a tuple with the CapturedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCapturedAt

`func (o *ReportHostMetrics200Response) SetCapturedAt(v time.Time)`

SetCapturedAt sets CapturedAt field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


