
# Device

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **deviceId** | **kotlin.String** |  |  |
| **deviceName** | **kotlin.String** |  |  |
| **online** | **kotlin.Boolean** | Whether the machine currently holds a live connection. |  |
| **loggedIn** | **kotlin.Boolean** | True only when the device is online *and* logged in, so a parked machine is not reported as reachable. |  |
| **remark** | **kotlin.String** | The owner&#39;s private note; empty when none is set. |  [optional] |
| **accessCode** | **kotlin.String** | The current access code. It rotates, so treat it as short-lived. |  [optional] |
| **os** | **kotlin.String** |  |  [optional] |
| **osVersion** | **kotlin.String** |  |  [optional] |
| **appVersion** | **kotlin.String** |  |  [optional] |
| **lastSeenAt** | **kotlin.String** | RFC3339 UTC, omitted when the device has never been seen. |  [optional] |



