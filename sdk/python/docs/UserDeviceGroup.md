# UserDeviceGroup


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **int** |  | 
**user_id** | **int** |  | 
**name** | **str** |  | 
**description** | **str** |  | 
**color** | **str** |  | 
**device_ids** | **List[str]** |  | 
**device_count** | **int** |  | 
**created_at** | **datetime** |  | 
**updated_at** | **datetime** |  | 

## Example

```python
from quickdesk_signaling.models.user_device_group import UserDeviceGroup

# TODO update the JSON string below
json = "{}"
# create an instance of UserDeviceGroup from a JSON string
user_device_group_instance = UserDeviceGroup.from_json(json)
# print the JSON string representation of the object
print(UserDeviceGroup.to_json())

# convert the object into a dict
user_device_group_dict = user_device_group_instance.to_dict()
# create an instance of UserDeviceGroup from a dict
user_device_group_from_dict = UserDeviceGroup.from_dict(user_device_group_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


