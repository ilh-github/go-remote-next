
# OfflineAlertInput

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **enabled** | **kotlin.Boolean** |  |  |
| **afterMinutes** | **kotlin.Int** | Omitting it (or sending 0) stores the default of 10. |  [optional] |
| **repeatMinutes** | **kotlin.Int** | 0 (the default) sends one alert per outage. |  [optional] |
| **snoozedUntil** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) | Silences alerts until this instant; null clears an existing snooze. |  [optional] |
| **maintenanceWindows** | [**kotlin.collections.List&lt;MaintenanceWindow&gt;**](MaintenanceWindow.md) | Recurring quiet periods evaluated in &#x60;maintenance_timezone&#x60;. |  [optional] |
| **maintenanceTimezone** | **kotlin.String** | IANA zone name for the windows; empty means the server&#39;s local zone. |  [optional] |



