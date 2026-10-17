# # MetricAlertInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**enabled** | **bool** |  |
**cpu_percent** | **int** | 0 disables the CPU threshold. |
**memory_percent** | **int** | 0 disables the memory threshold. |
**disk_percent** | **int** | 0 disables the disk threshold. |
**sustained_minutes** | **int** | 0 alerts on the first over-threshold sample. |
**repeat_minutes** | **int** | 0 sends one alert per continuous breach. |
**snoozed_until** | **\DateTime** |  | [optional]

[[Back to Model list]](../../README.md#models) [[Back to API list]](../../README.md#endpoints) [[Back to README]](../../README.md)
