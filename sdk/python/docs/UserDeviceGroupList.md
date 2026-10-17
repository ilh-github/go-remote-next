# UserDeviceGroupList


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[UserDeviceGroup]**](UserDeviceGroup.md) |  | 

## Example

```python
from quickdesk_signaling.models.user_device_group_list import UserDeviceGroupList

# TODO update the JSON string below
json = "{}"
# create an instance of UserDeviceGroupList from a JSON string
user_device_group_list_instance = UserDeviceGroupList.from_json(json)
# print the JSON string representation of the object
print(UserDeviceGroupList.to_json())

# convert the object into a dict
user_device_group_list_dict = user_device_group_list_instance.to_dict()
# create an instance of UserDeviceGroupList from a dict
user_device_group_list_from_dict = UserDeviceGroupList.from_dict(user_device_group_list_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


