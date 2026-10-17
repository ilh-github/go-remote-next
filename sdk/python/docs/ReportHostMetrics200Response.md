# ReportHostMetrics200Response


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**device_id** | **str** |  | 
**captured_at** | **datetime** |  | 

## Example

```python
from quickdesk_signaling.models.report_host_metrics200_response import ReportHostMetrics200Response

# TODO update the JSON string below
json = "{}"
# create an instance of ReportHostMetrics200Response from a JSON string
report_host_metrics200_response_instance = ReportHostMetrics200Response.from_json(json)
# print the JSON string representation of the object
print(ReportHostMetrics200Response.to_json())

# convert the object into a dict
report_host_metrics200_response_dict = report_host_metrics200_response_instance.to_dict()
# create an instance of ReportHostMetrics200Response from a dict
report_host_metrics200_response_from_dict = ReportHostMetrics200Response.from_dict(report_host_metrics200_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


