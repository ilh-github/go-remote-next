# AccessCodeVerifyResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**signal_token** | **str** |  | 
**expires_at** | **datetime** |  | 

## Example

```python
from quickdesk_signaling.models.access_code_verify_response import AccessCodeVerifyResponse

# TODO update the JSON string below
json = "{}"
# create an instance of AccessCodeVerifyResponse from a JSON string
access_code_verify_response_instance = AccessCodeVerifyResponse.from_json(json)
# print the JSON string representation of the object
print(AccessCodeVerifyResponse.to_json())

# convert the object into a dict
access_code_verify_response_dict = access_code_verify_response_instance.to_dict()
# create an instance of AccessCodeVerifyResponse from a dict
access_code_verify_response_from_dict = AccessCodeVerifyResponse.from_dict(access_code_verify_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


