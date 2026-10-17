
# MetricAlertInput

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **enabled** | **kotlin.Boolean** |  |  |
| **cpuPercent** | **kotlin.Int** | 0 disables the CPU threshold. |  |
| **memoryPercent** | **kotlin.Int** | 0 disables the memory threshold. |  |
| **diskPercent** | **kotlin.Int** | 0 disables the disk threshold. |  |
| **sustainedMinutes** | **kotlin.Int** | 0 alerts on the first over-threshold sample. |  |
| **repeatMinutes** | **kotlin.Int** | 0 sends one alert per continuous breach. |  |
| **snoozedUntil** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  [optional] |



