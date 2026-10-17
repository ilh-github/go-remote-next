# DeviceGroupsApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**createMyDeviceGroup**](DeviceGroupsApi.md#createMyDeviceGroup) | **POST** /v1/me/device-groups | Create a private address-book group |
| [**deleteMyDeviceGroup**](DeviceGroupsApi.md#deleteMyDeviceGroup) | **DELETE** /v1/me/device-groups/{group_id} | Delete one of the caller&#39;s address-book groups |
| [**listMyDeviceGroups**](DeviceGroupsApi.md#listMyDeviceGroups) | **GET** /v1/me/device-groups | The caller&#39;s private address-book groups |
| [**replaceMyDeviceGroupDevices**](DeviceGroupsApi.md#replaceMyDeviceGroupDevices) | **PUT** /v1/me/device-groups/{group_id}/devices | Replace the devices in one of the caller&#39;s groups |
| [**updateMyDeviceGroup**](DeviceGroupsApi.md#updateMyDeviceGroup) | **PATCH** /v1/me/device-groups/{group_id} | Update one of the caller&#39;s address-book groups |


<a id="createMyDeviceGroup"></a>
# **createMyDeviceGroup**
> UserDeviceGroup createMyDeviceGroup(userDeviceGroupInput)

Create a private address-book group

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DeviceGroupsApi()
val userDeviceGroupInput : UserDeviceGroupInput =  // UserDeviceGroupInput | 
try {
    val result : UserDeviceGroup = apiInstance.createMyDeviceGroup(userDeviceGroupInput)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DeviceGroupsApi#createMyDeviceGroup")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DeviceGroupsApi#createMyDeviceGroup")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **userDeviceGroupInput** | [**UserDeviceGroupInput**](UserDeviceGroupInput.md)|  | |

### Return type

[**UserDeviceGroup**](UserDeviceGroup.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

<a id="deleteMyDeviceGroup"></a>
# **deleteMyDeviceGroup**
> DeleteMyDeviceGroup200Response deleteMyDeviceGroup(groupId)

Delete one of the caller&#39;s address-book groups

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DeviceGroupsApi()
val groupId : kotlin.Long = 789 // kotlin.Long | The numeric id of one of the caller's private device groups.
try {
    val result : DeleteMyDeviceGroup200Response = apiInstance.deleteMyDeviceGroup(groupId)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DeviceGroupsApi#deleteMyDeviceGroup")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DeviceGroupsApi#deleteMyDeviceGroup")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **groupId** | **kotlin.Long**| The numeric id of one of the caller&#39;s private device groups. | |

### Return type

[**DeleteMyDeviceGroup200Response**](DeleteMyDeviceGroup200Response.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

<a id="listMyDeviceGroups"></a>
# **listMyDeviceGroups**
> UserDeviceGroupList listMyDeviceGroups()

The caller&#39;s private address-book groups

Owner-scoped groups are separate from the administrator-wide device inventory under &#x60;/v1/admin/groups&#x60;. Each returned group carries its current &#x60;device_ids&#x60;, so a client can render filters and membership without issuing one request per group. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DeviceGroupsApi()
try {
    val result : UserDeviceGroupList = apiInstance.listMyDeviceGroups()
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DeviceGroupsApi#listMyDeviceGroups")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DeviceGroupsApi#listMyDeviceGroups")
    e.printStackTrace()
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**UserDeviceGroupList**](UserDeviceGroupList.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

<a id="replaceMyDeviceGroupDevices"></a>
# **replaceMyDeviceGroupDevices**
> UserDeviceGroup replaceMyDeviceGroupDevices(groupId, userDeviceGroupDevicesInput)

Replace the devices in one of the caller&#39;s groups

Replaces the whole membership atomically. Every device must currently belong to the caller; an unknown or foreign device is rejected with &#x60;403 DEVICE_NOT_OWNED&#x60;. Sending an empty &#x60;device_ids&#x60; array clears the group. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DeviceGroupsApi()
val groupId : kotlin.Long = 789 // kotlin.Long | The numeric id of one of the caller's private device groups.
val userDeviceGroupDevicesInput : UserDeviceGroupDevicesInput =  // UserDeviceGroupDevicesInput | 
try {
    val result : UserDeviceGroup = apiInstance.replaceMyDeviceGroupDevices(groupId, userDeviceGroupDevicesInput)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DeviceGroupsApi#replaceMyDeviceGroupDevices")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DeviceGroupsApi#replaceMyDeviceGroupDevices")
    e.printStackTrace()
}
```

### Parameters
| **groupId** | **kotlin.Long**| The numeric id of one of the caller&#39;s private device groups. | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **userDeviceGroupDevicesInput** | [**UserDeviceGroupDevicesInput**](UserDeviceGroupDevicesInput.md)|  | |

### Return type

[**UserDeviceGroup**](UserDeviceGroup.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

<a id="updateMyDeviceGroup"></a>
# **updateMyDeviceGroup**
> UserDeviceGroup updateMyDeviceGroup(groupId, userDeviceGroupPatch)

Update one of the caller&#39;s address-book groups

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DeviceGroupsApi()
val groupId : kotlin.Long = 789 // kotlin.Long | The numeric id of one of the caller's private device groups.
val userDeviceGroupPatch : UserDeviceGroupPatch =  // UserDeviceGroupPatch | 
try {
    val result : UserDeviceGroup = apiInstance.updateMyDeviceGroup(groupId, userDeviceGroupPatch)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DeviceGroupsApi#updateMyDeviceGroup")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DeviceGroupsApi#updateMyDeviceGroup")
    e.printStackTrace()
}
```

### Parameters
| **groupId** | **kotlin.Long**| The numeric id of one of the caller&#39;s private device groups. | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **userDeviceGroupPatch** | [**UserDeviceGroupPatch**](UserDeviceGroupPatch.md)|  | |

### Return type

[**UserDeviceGroup**](UserDeviceGroup.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

