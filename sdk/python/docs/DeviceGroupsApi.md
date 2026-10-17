# quickdesk_signaling.DeviceGroupsApi

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**create_my_device_group**](DeviceGroupsApi.md#create_my_device_group) | **POST** /v1/me/device-groups | Create a private address-book group
[**delete_my_device_group**](DeviceGroupsApi.md#delete_my_device_group) | **DELETE** /v1/me/device-groups/{group_id} | Delete one of the caller&#39;s address-book groups
[**list_my_device_groups**](DeviceGroupsApi.md#list_my_device_groups) | **GET** /v1/me/device-groups | The caller&#39;s private address-book groups
[**replace_my_device_group_devices**](DeviceGroupsApi.md#replace_my_device_group_devices) | **PUT** /v1/me/device-groups/{group_id}/devices | Replace the devices in one of the caller&#39;s groups
[**update_my_device_group**](DeviceGroupsApi.md#update_my_device_group) | **PATCH** /v1/me/device-groups/{group_id} | Update one of the caller&#39;s address-book groups


# **create_my_device_group**
> UserDeviceGroup create_my_device_group(user_device_group_input)

Create a private address-book group

### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.user_device_group import UserDeviceGroup
from quickdesk_signaling.models.user_device_group_input import UserDeviceGroupInput
from quickdesk_signaling.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://your-signaling-host
# See configuration.py for a list of all supported configuration parameters.
configuration = quickdesk_signaling.Configuration(
    host = "https://your-signaling-host"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Configure Bearer authorization (opaque): BearerAuth
configuration = quickdesk_signaling.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with quickdesk_signaling.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = quickdesk_signaling.DeviceGroupsApi(api_client)
    user_device_group_input = quickdesk_signaling.UserDeviceGroupInput() # UserDeviceGroupInput | 

    try:
        # Create a private address-book group
        api_response = api_instance.create_my_device_group(user_device_group_input)
        print("The response of DeviceGroupsApi->create_my_device_group:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DeviceGroupsApi->create_my_device_group: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **user_device_group_input** | [**UserDeviceGroupInput**](UserDeviceGroupInput.md)|  | 

### Return type

[**UserDeviceGroup**](UserDeviceGroup.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**201** | The created group. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |
**409** | The caller already has a group with that name. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **delete_my_device_group**
> DeleteMyDeviceGroup200Response delete_my_device_group(group_id)

Delete one of the caller's address-book groups

### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.delete_my_device_group200_response import DeleteMyDeviceGroup200Response
from quickdesk_signaling.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://your-signaling-host
# See configuration.py for a list of all supported configuration parameters.
configuration = quickdesk_signaling.Configuration(
    host = "https://your-signaling-host"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Configure Bearer authorization (opaque): BearerAuth
configuration = quickdesk_signaling.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with quickdesk_signaling.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = quickdesk_signaling.DeviceGroupsApi(api_client)
    group_id = 56 # int | The numeric id of one of the caller's private device groups.

    try:
        # Delete one of the caller's address-book groups
        api_response = api_instance.delete_my_device_group(group_id)
        print("The response of DeviceGroupsApi->delete_my_device_group:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DeviceGroupsApi->delete_my_device_group: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **group_id** | **int**| The numeric id of one of the caller&#39;s private device groups. | 

### Return type

[**DeleteMyDeviceGroup200Response**](DeleteMyDeviceGroup200Response.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The group was deleted. |  -  |
**401** | Missing or expired credentials. |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_my_device_groups**
> UserDeviceGroupList list_my_device_groups()

The caller's private address-book groups

Owner-scoped groups are separate from the administrator-wide device
inventory under `/v1/admin/groups`. Each returned group carries its
current `device_ids`, so a client can render filters and membership
without issuing one request per group.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.user_device_group_list import UserDeviceGroupList
from quickdesk_signaling.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://your-signaling-host
# See configuration.py for a list of all supported configuration parameters.
configuration = quickdesk_signaling.Configuration(
    host = "https://your-signaling-host"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Configure Bearer authorization (opaque): BearerAuth
configuration = quickdesk_signaling.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with quickdesk_signaling.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = quickdesk_signaling.DeviceGroupsApi(api_client)

    try:
        # The caller's private address-book groups
        api_response = api_instance.list_my_device_groups()
        print("The response of DeviceGroupsApi->list_my_device_groups:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DeviceGroupsApi->list_my_device_groups: %s\n" % e)
```



### Parameters

This endpoint does not need any parameter.

### Return type

[**UserDeviceGroupList**](UserDeviceGroupList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The caller&#39;s groups. |  -  |
**401** | Missing or expired credentials. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **replace_my_device_group_devices**
> UserDeviceGroup replace_my_device_group_devices(group_id, user_device_group_devices_input)

Replace the devices in one of the caller's groups

Replaces the whole membership atomically. Every device must currently
belong to the caller; an unknown or foreign device is rejected with
`403 DEVICE_NOT_OWNED`. Sending an empty `device_ids` array clears the
group.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.user_device_group import UserDeviceGroup
from quickdesk_signaling.models.user_device_group_devices_input import UserDeviceGroupDevicesInput
from quickdesk_signaling.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://your-signaling-host
# See configuration.py for a list of all supported configuration parameters.
configuration = quickdesk_signaling.Configuration(
    host = "https://your-signaling-host"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Configure Bearer authorization (opaque): BearerAuth
configuration = quickdesk_signaling.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with quickdesk_signaling.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = quickdesk_signaling.DeviceGroupsApi(api_client)
    group_id = 56 # int | The numeric id of one of the caller's private device groups.
    user_device_group_devices_input = quickdesk_signaling.UserDeviceGroupDevicesInput() # UserDeviceGroupDevicesInput | 

    try:
        # Replace the devices in one of the caller's groups
        api_response = api_instance.replace_my_device_group_devices(group_id, user_device_group_devices_input)
        print("The response of DeviceGroupsApi->replace_my_device_group_devices:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DeviceGroupsApi->replace_my_device_group_devices: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **group_id** | **int**| The numeric id of one of the caller&#39;s private device groups. | 
 **user_device_group_devices_input** | [**UserDeviceGroupDevicesInput**](UserDeviceGroupDevicesInput.md)|  | 

### Return type

[**UserDeviceGroup**](UserDeviceGroup.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The updated group. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |
**403** | Authenticated but not allowed (includes a rejected access code). |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **update_my_device_group**
> UserDeviceGroup update_my_device_group(group_id, user_device_group_patch)

Update one of the caller's address-book groups

### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.user_device_group import UserDeviceGroup
from quickdesk_signaling.models.user_device_group_patch import UserDeviceGroupPatch
from quickdesk_signaling.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://your-signaling-host
# See configuration.py for a list of all supported configuration parameters.
configuration = quickdesk_signaling.Configuration(
    host = "https://your-signaling-host"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Configure Bearer authorization (opaque): BearerAuth
configuration = quickdesk_signaling.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with quickdesk_signaling.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = quickdesk_signaling.DeviceGroupsApi(api_client)
    group_id = 56 # int | The numeric id of one of the caller's private device groups.
    user_device_group_patch = quickdesk_signaling.UserDeviceGroupPatch() # UserDeviceGroupPatch | 

    try:
        # Update one of the caller's address-book groups
        api_response = api_instance.update_my_device_group(group_id, user_device_group_patch)
        print("The response of DeviceGroupsApi->update_my_device_group:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DeviceGroupsApi->update_my_device_group: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **group_id** | **int**| The numeric id of one of the caller&#39;s private device groups. | 
 **user_device_group_patch** | [**UserDeviceGroupPatch**](UserDeviceGroupPatch.md)|  | 

### Return type

[**UserDeviceGroup**](UserDeviceGroup.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The updated group. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |
**404** | The addressed resource does not exist. |  -  |
**409** | The caller already has a group with that name. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

