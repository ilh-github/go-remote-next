# MetricAlertState


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**exceeded_since** | **datetime** |  | [optional] 
**last_alerted_at** | **datetime** |  | [optional] 

## Example

```python
from quickdesk_signaling.models.metric_alert_state import MetricAlertState

# TODO update the JSON string below
json = "{}"
# create an instance of MetricAlertState from a JSON string
metric_alert_state_instance = MetricAlertState.from_json(json)
# print the JSON string representation of the object
print(MetricAlertState.to_json())

# convert the object into a dict
metric_alert_state_dict = metric_alert_state_instance.to_dict()
# create an instance of MetricAlertState from a dict
metric_alert_state_from_dict = MetricAlertState.from_dict(metric_alert_state_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


