# # OfflineAlertInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**enabled** | **bool** |  |
**after_minutes** | **int** | Omitting it (or sending 0) stores the default of 10. | [optional]
**repeat_minutes** | **int** | 0 (the default) sends one alert per outage. | [optional]
**snoozed_until** | **\DateTime** | Silences alerts until this instant; null clears an existing snooze. | [optional]
**maintenance_windows** | [**\OpenAPI\Client\Model\MaintenanceWindow[]**](MaintenanceWindow.md) | Recurring quiet periods evaluated in &#x60;maintenance_timezone&#x60;. | [optional]
**maintenance_timezone** | **string** | IANA zone name for the windows; empty means the server&#39;s local zone. | [optional]

[[Back to Model list]](../../README.md#models) [[Back to API list]](../../README.md#endpoints) [[Back to README]](../../README.md)
