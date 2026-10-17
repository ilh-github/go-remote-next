# HostMetricReport


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
from quickdesk_signaling.models.host_metric_report import HostMetricReport

# TODO update the JSON string below
json = "{}"
# create an instance of HostMetricReport from a JSON string
host_metric_report_instance = HostMetricReport.from_json(json)
# print the JSON string representation of the object
print(HostMetricReport.to_json())

# convert the object into a dict
host_metric_report_dict = host_metric_report_instance.to_dict()
# create an instance of HostMetricReport from a dict
host_metric_report_from_dict = HostMetricReport.from_dict(host_metric_report_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


