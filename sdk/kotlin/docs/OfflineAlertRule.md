
# OfflineAlertRule

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **deviceId** | **kotlin.String** |  |  |
| **enabled** | **kotlin.Boolean** |  |  |
| **afterMinutes** | **kotlin.Int** |  |  |
| **repeatMinutes** | **kotlin.Int** |  |  |
| **maintenanceWindows** | [**kotlin.collections.List&lt;MaintenanceWindow&gt;**](MaintenanceWindow.md) |  |  |
| **inMaintenanceWindow** | **kotlin.Boolean** | Computed at read time, so a client can explain a quiet period without re-implementing the schedule. |  |
| **snoozedUntil** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  [optional] |
| **maintenanceTimezone** | **kotlin.String** |  |  [optional] |
| **offlineSince** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) | The evaluator&#39;s episode state; present only while the device is in an outage. |  [optional] |
| **lastAlertedAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  [optional] |
| **updatedAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  [optional] |



