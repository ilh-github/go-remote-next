# AccessCodeVerifyResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**SignalToken** | **string** |  | 
**ExpiresAt** | **time.Time** |  | 

## Methods

### NewAccessCodeVerifyResponse

`func NewAccessCodeVerifyResponse(signalToken string, expiresAt time.Time, ) *AccessCodeVerifyResponse`

NewAccessCodeVerifyResponse instantiates a new AccessCodeVerifyResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAccessCodeVerifyResponseWithDefaults

`func NewAccessCodeVerifyResponseWithDefaults() *AccessCodeVerifyResponse`

NewAccessCodeVerifyResponseWithDefaults instantiates a new AccessCodeVerifyResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSignalToken

`func (o *AccessCodeVerifyResponse) GetSignalToken() string`

GetSignalToken returns the SignalToken field if non-nil, zero value otherwise.

### GetSignalTokenOk

`func (o *AccessCodeVerifyResponse) GetSignalTokenOk() (*string, bool)`

GetSignalTokenOk returns a tuple with the SignalToken field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSignalToken

`func (o *AccessCodeVerifyResponse) SetSignalToken(v string)`

SetSignalToken sets SignalToken field to given value.


### GetExpiresAt

`func (o *AccessCodeVerifyResponse) GetExpiresAt() time.Time`

GetExpiresAt returns the ExpiresAt field if non-nil, zero value otherwise.

### GetExpiresAtOk

`func (o *AccessCodeVerifyResponse) GetExpiresAtOk() (*time.Time, bool)`

GetExpiresAtOk returns a tuple with the ExpiresAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExpiresAt

`func (o *AccessCodeVerifyResponse) SetExpiresAt(v time.Time)`

SetExpiresAt sets ExpiresAt field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


