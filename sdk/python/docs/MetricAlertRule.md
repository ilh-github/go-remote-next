# MetricAlertRule


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**device_id** | **str** |  | 
**enabled** | **bool** |  | 
**cpu_percent** | **int** |  | 
**memory_percent** | **int** |  | 
**disk_percent** | **int** |  | 
**sustained_minutes** | **int** |  | 
**repeat_minutes** | **int** |  | 
**snoozed_until** | **datetime** |  | [optional] 
**current** | [**MetricSnapshot**](MetricSnapshot.md) |  | [optional] 
**states** | [**MetricAlertStates**](MetricAlertStates.md) |  | 
**updated_at** | **datetime** |  | [optional] 

## Example

```python
from quickdesk_signaling.models.metric_alert_rule import MetricAlertRule

# TODO update the JSON string below
json = "{}"
# create an instance of MetricAlertRule from a JSON string
metric_alert_rule_instance = MetricAlertRule.from_json(json)
# print the JSON string representation of the object
print(MetricAlertRule.to_json())

# convert the object into a dict
metric_alert_rule_dict = metric_alert_rule_instance.to_dict()
# create an instance of MetricAlertRule from a dict
metric_alert_rule_from_dict = MetricAlertRule.from_dict(metric_alert_rule_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


