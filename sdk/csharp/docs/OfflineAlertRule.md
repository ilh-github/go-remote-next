# QuickDesk.Signaling.Client.Model.OfflineAlertRule

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DeviceId** | **string** |  | 
**Enabled** | **bool** |  | 
**AfterMinutes** | **int** |  | 
**RepeatMinutes** | **int** |  | 
**SnoozedUntil** | **DateTime?** |  | [optional] 
**MaintenanceWindows** | [**List&lt;MaintenanceWindow&gt;**](MaintenanceWindow.md) |  | 
**MaintenanceTimezone** | **string** |  | [optional] 
**InMaintenanceWindow** | **bool** | Computed at read time, so a client can explain a quiet period without re-implementing the schedule. | 
**OfflineSince** | **DateTime?** | The evaluator&#39;s episode state; present only while the device is in an outage. | [optional] 
**LastAlertedAt** | **DateTime?** |  | [optional] 
**UpdatedAt** | **DateTime?** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

