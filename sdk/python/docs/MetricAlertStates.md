# MetricAlertStates


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**cpu** | [**MetricAlertState**](MetricAlertState.md) |  | 
**memory** | [**MetricAlertState**](MetricAlertState.md) |  | 
**disk** | [**MetricAlertState**](MetricAlertState.md) |  | 

## Example

```python
from quickdesk_signaling.models.metric_alert_states import MetricAlertStates

# TODO update the JSON string below
json = "{}"
# create an instance of MetricAlertStates from a JSON string
metric_alert_states_instance = MetricAlertStates.from_json(json)
# print the JSON string representation of the object
print(MetricAlertStates.to_json())

# convert the object into a dict
metric_alert_states_dict = metric_alert_states_instance.to_dict()
# create an instance of MetricAlertStates from a dict
metric_alert_states_from_dict = MetricAlertStates.from_dict(metric_alert_states_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


