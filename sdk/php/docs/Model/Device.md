# # Device

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**device_id** | **string** |  |
**device_name** | **string** |  |
**remark** | **string** | The owner&#39;s private note; empty when none is set. | [optional]
**online** | **bool** | Whether the machine currently holds a live connection. |
**logged_in** | **bool** | True only when the device is online *and* logged in, so a parked machine is not reported as reachable. |
**access_code** | **string** | The current access code. It rotates, so treat it as short-lived. | [optional]
**os** | **string** |  | [optional]
**os_version** | **string** |  | [optional]
**app_version** | **string** |  | [optional]
**last_seen_at** | **string** | RFC3339 UTC, omitted when the device has never been seen. | [optional]

[[Back to Model list]](../../README.md#models) [[Back to API list]](../../README.md#endpoints) [[Back to README]](../../README.md)
