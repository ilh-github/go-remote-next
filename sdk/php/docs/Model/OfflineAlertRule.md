# # OfflineAlertRule

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**device_id** | **string** |  |
**enabled** | **bool** |  |
**after_minutes** | **int** |  |
**repeat_minutes** | **int** |  |
**snoozed_until** | **\DateTime** |  | [optional]
**maintenance_windows** | [**\OpenAPI\Client\Model\MaintenanceWindow[]**](MaintenanceWindow.md) |  |
**maintenance_timezone** | **string** |  | [optional]
**in_maintenance_window** | **bool** | Computed at read time, so a client can explain a quiet period without re-implementing the schedule. |
**offline_since** | **\DateTime** | The evaluator&#39;s episode state; present only while the device is in an outage. | [optional]
**last_alerted_at** | **\DateTime** |  | [optional]
**updated_at** | **\DateTime** |  | [optional]

[[Back to Model list]](../../README.md#models) [[Back to API list]](../../README.md#endpoints) [[Back to README]](../../README.md)
