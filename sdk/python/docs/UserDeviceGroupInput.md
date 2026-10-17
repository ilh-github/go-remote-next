# UserDeviceGroupInput


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** |  | 
**description** | **str** |  | [optional] 
**color** | **str** | Optional display color, such as &#x60;#1f6feb&#x60;. | [optional] 

## Example

```python
from quickdesk_signaling.models.user_device_group_input import UserDeviceGroupInput

# TODO update the JSON string below
json = "{}"
# create an instance of UserDeviceGroupInput from a JSON string
user_device_group_input_instance = UserDeviceGroupInput.from_json(json)
# print the JSON string representation of the object
print(UserDeviceGroupInput.to_json())

# convert the object into a dict
user_device_group_input_dict = user_device_group_input_instance.to_dict()
# create an instance of UserDeviceGroupInput from a dict
user_device_group_input_from_dict = UserDeviceGroupInput.from_dict(user_device_group_input_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


