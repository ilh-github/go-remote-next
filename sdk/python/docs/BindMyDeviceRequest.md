# BindMyDeviceRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**device_id** | **str** |  | 

## Example

```python
from quickdesk_signaling.models.bind_my_device_request import BindMyDeviceRequest

# TODO update the JSON string below
json = "{}"
# create an instance of BindMyDeviceRequest from a JSON string
bind_my_device_request_instance = BindMyDeviceRequest.from_json(json)
# print the JSON string representation of the object
print(BindMyDeviceRequest.to_json())

# convert the object into a dict
bind_my_device_request_dict = bind_my_device_request_instance.to_dict()
# create an instance of BindMyDeviceRequest from a dict
bind_my_device_request_from_dict = BindMyDeviceRequest.from_dict(bind_my_device_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


