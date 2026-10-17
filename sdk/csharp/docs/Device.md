# QuickDesk.Signaling.Client.Model.Device

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DeviceId** | **string** |  | 
**DeviceName** | **string** |  | 
**Remark** | **string** | The owner&#39;s private note; empty when none is set. | [optional] 
**Online** | **bool** | Whether the machine currently holds a live connection. | 
**LoggedIn** | **bool** | True only when the device is online *and* logged in, so a parked machine is not reported as reachable. | 
**AccessCode** | **string** | The current access code. It rotates, so treat it as short-lived. | [optional] 
**Os** | **string** |  | [optional] 
**OsVersion** | **string** |  | [optional] 
**AppVersion** | **string** |  | [optional] 
**LastSeenAt** | **string** | RFC3339 UTC, omitted when the device has never been seen. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

