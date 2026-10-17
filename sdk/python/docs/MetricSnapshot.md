# MetricSnapshot


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**captured_at** | **datetime** |  | 
**cpu_percent** | **float** |  | 
**memory_percent** | **float** |  | 
**disk_percent** | **float** |  | 
**disk_mount** | **str** |  | 

## Example

```python
from quickdesk_signaling.models.metric_snapshot import MetricSnapshot

# TODO update the JSON string below
json = "{}"
# create an instance of MetricSnapshot from a JSON string
metric_snapshot_instance = MetricSnapshot.from_json(json)
# print the JSON string representation of the object
print(MetricSnapshot.to_json())

# convert the object into a dict
metric_snapshot_dict = metric_snapshot_instance.to_dict()
# create an instance of MetricSnapshot from a dict
metric_snapshot_from_dict = MetricSnapshot.from_dict(metric_snapshot_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


