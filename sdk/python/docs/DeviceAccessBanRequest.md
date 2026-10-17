# DeviceAccessBanRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**source** | **str** | Exact IPv4/IPv6 address or CIDR block to deny permanently. | 

## Example

```python
from quickdesk_signaling.models.device_access_ban_request import DeviceAccessBanRequest

# TODO update the JSON string below
json = "{}"
# create an instance of DeviceAccessBanRequest from a JSON string
device_access_ban_request_instance = DeviceAccessBanRequest.from_json(json)
# print the JSON string representation of the object
print(DeviceAccessBanRequest.to_json())

# convert the object into a dict
device_access_ban_request_dict = device_access_ban_request_instance.to_dict()
# create an instance of DeviceAccessBanRequest from a dict
device_access_ban_request_from_dict = DeviceAccessBanRequest.from_dict(device_access_ban_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


