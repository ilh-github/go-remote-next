# QuickDesk.Signaling.Client.Model.OfflineAlertInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Enabled** | **bool** |  | 
**AfterMinutes** | **int** | Omitting it (or sending 0) stores the default of 10. | [optional] 
**RepeatMinutes** | **int** | 0 (the default) sends one alert per outage. | [optional] 
**SnoozedUntil** | **DateTime?** | Silences alerts until this instant; null clears an existing snooze. | [optional] 
**MaintenanceWindows** | [**List&lt;MaintenanceWindow&gt;**](MaintenanceWindow.md) | Recurring quiet periods evaluated in &#x60;maintenance_timezone&#x60;. | [optional] 
**MaintenanceTimezone** | **string** | IANA zone name for the windows; empty means the server&#39;s local zone. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

