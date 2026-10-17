# UserDeviceGroupPatch


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** |  | [optional] 
**description** | **str** |  | [optional] 
**color** | **str** |  | [optional] 

## Example

```python
from quickdesk_signaling.models.user_device_group_patch import UserDeviceGroupPatch

# TODO update the JSON string below
json = "{}"
# create an instance of UserDeviceGroupPatch from a JSON string
user_device_group_patch_instance = UserDeviceGroupPatch.from_json(json)
# print the JSON string representation of the object
print(UserDeviceGroupPatch.to_json())

# convert the object into a dict
user_device_group_patch_dict = user_device_group_patch_instance.to_dict()
# create an instance of UserDeviceGroupPatch from a dict
user_device_group_patch_from_dict = UserDeviceGroupPatch.from_dict(user_device_group_patch_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


