# QuickDesk.Signaling.Client.Model.MetricAlertInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Enabled** | **bool** |  | 
**CpuPercent** | **int** | 0 disables the CPU threshold. | 
**MemoryPercent** | **int** | 0 disables the memory threshold. | 
**DiskPercent** | **int** | 0 disables the disk threshold. | 
**SustainedMinutes** | **int** | 0 alerts on the first over-threshold sample. | 
**RepeatMinutes** | **int** | 0 sends one alert per continuous breach. | 
**SnoozedUntil** | **DateTime?** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

