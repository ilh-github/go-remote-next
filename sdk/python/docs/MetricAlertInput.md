# MetricAlertInput


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**enabled** | **bool** |  | 
**cpu_percent** | **int** | 0 disables the CPU threshold. | 
**memory_percent** | **int** | 0 disables the memory threshold. | 
**disk_percent** | **int** | 0 disables the disk threshold. | 
**sustained_minutes** | **int** | 0 alerts on the first over-threshold sample. | 
**repeat_minutes** | **int** | 0 sends one alert per continuous breach. | 
**snoozed_until** | **datetime** |  | [optional] 

## Example

```python
from quickdesk_signaling.models.metric_alert_input import MetricAlertInput

# TODO update the JSON string below
json = "{}"
# create an instance of MetricAlertInput from a JSON string
metric_alert_input_instance = MetricAlertInput.from_json(json)
# print the JSON string representation of the object
print(MetricAlertInput.to_json())

# convert the object into a dict
metric_alert_input_dict = metric_alert_input_instance.to_dict()
# create an instance of MetricAlertInput from a dict
metric_alert_input_from_dict = MetricAlertInput.from_dict(metric_alert_input_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


