# UserDeviceGroupDevicesInput


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**device_ids** | **List[str]** |  | 

## Example

```python
from quickdesk_signaling.models.user_device_group_devices_input import UserDeviceGroupDevicesInput

# TODO update the JSON string below
json = "{}"
# create an instance of UserDeviceGroupDevicesInput from a JSON string
user_device_group_devices_input_instance = UserDeviceGroupDevicesInput.from_json(json)
# print the JSON string representation of the object
print(UserDeviceGroupDevicesInput.to_json())

# convert the object into a dict
user_device_group_devices_input_dict = user_device_group_devices_input_instance.to_dict()
# create an instance of UserDeviceGroupDevicesInput from a dict
user_device_group_devices_input_from_dict = UserDeviceGroupDevicesInput.from_dict(user_device_group_devices_input_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


