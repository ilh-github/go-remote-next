# DeviceGroupsApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
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
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.DeviceGroupsApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("https://your-signaling-host");
    
    // Configure API key authorization: ApiKeyAuth
    ApiKeyAuth ApiKeyAuth = (ApiKeyAuth) defaultClient.getAuthentication("ApiKeyAuth");
    ApiKeyAuth.setApiKey("YOUR API KEY");
    // Uncomment the following line to set a prefix for the API key, e.g. "Token" (defaults to null)
    //ApiKeyAuth.setApiKeyPrefix("Token");

    // Configure HTTP bearer authorization: BearerAuth
    HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
    BearerAuth.setBearerToken("BEARER TOKEN");

    DeviceGroupsApi apiInstance = new DeviceGroupsApi(defaultClient);
    UserDeviceGroupInput userDeviceGroupInput = new UserDeviceGroupInput(); // UserDeviceGroupInput | 
    try {
      UserDeviceGroup result = apiInstance.createMyDeviceGroup(userDeviceGroupInput);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling DeviceGroupsApi#createMyDeviceGroup");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters

| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **userDeviceGroupInput** | [**UserDeviceGroupInput**](UserDeviceGroupInput.md)|  | |

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
| **201** | The created group. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **409** | The caller already has a group with that name. |  -  |

<a id="deleteMyDeviceGroup"></a>
# **deleteMyDeviceGroup**
> DeleteMyDeviceGroup200Response deleteMyDeviceGroup(groupId)

Delete one of the caller&#39;s address-book groups

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.DeviceGroupsApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("https://your-signaling-host");
    
    // Configure API key authorization: ApiKeyAuth
    ApiKeyAuth ApiKeyAuth = (ApiKeyAuth) defaultClient.getAuthentication("ApiKeyAuth");
    ApiKeyAuth.setApiKey("YOUR API KEY");
    // Uncomment the following line to set a prefix for the API key, e.g. "Token" (defaults to null)
    //ApiKeyAuth.setApiKeyPrefix("Token");

    // Configure HTTP bearer authorization: BearerAuth
    HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
    BearerAuth.setBearerToken("BEARER TOKEN");

    DeviceGroupsApi apiInstance = new DeviceGroupsApi(defaultClient);
    Long groupId = 56L; // Long | The numeric id of one of the caller's private device groups.
    try {
      DeleteMyDeviceGroup200Response result = apiInstance.deleteMyDeviceGroup(groupId);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling DeviceGroupsApi#deleteMyDeviceGroup");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters

| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **groupId** | **Long**| The numeric id of one of the caller&#39;s private device groups. | |

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
| **200** | The group was deleted. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="listMyDeviceGroups"></a>
# **listMyDeviceGroups**
> UserDeviceGroupList listMyDeviceGroups()

The caller&#39;s private address-book groups

Owner-scoped groups are separate from the administrator-wide device inventory under &#x60;/v1/admin/groups&#x60;. Each returned group carries its current &#x60;device_ids&#x60;, so a client can render filters and membership without issuing one request per group. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.DeviceGroupsApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("https://your-signaling-host");
    
    // Configure API key authorization: ApiKeyAuth
    ApiKeyAuth ApiKeyAuth = (ApiKeyAuth) defaultClient.getAuthentication("ApiKeyAuth");
    ApiKeyAuth.setApiKey("YOUR API KEY");
    // Uncomment the following line to set a prefix for the API key, e.g. "Token" (defaults to null)
    //ApiKeyAuth.setApiKeyPrefix("Token");

    // Configure HTTP bearer authorization: BearerAuth
    HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
    BearerAuth.setBearerToken("BEARER TOKEN");

    DeviceGroupsApi apiInstance = new DeviceGroupsApi(defaultClient);
    try {
      UserDeviceGroupList result = apiInstance.listMyDeviceGroups();
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling DeviceGroupsApi#listMyDeviceGroups");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
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
| **200** | The caller&#39;s groups. |  -  |
| **401** | Missing or expired credentials. |  -  |

<a id="replaceMyDeviceGroupDevices"></a>
# **replaceMyDeviceGroupDevices**
> UserDeviceGroup replaceMyDeviceGroupDevices(groupId, userDeviceGroupDevicesInput)

Replace the devices in one of the caller&#39;s groups

Replaces the whole membership atomically. Every device must currently belong to the caller; an unknown or foreign device is rejected with &#x60;403 DEVICE_NOT_OWNED&#x60;. Sending an empty &#x60;device_ids&#x60; array clears the group. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.DeviceGroupsApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("https://your-signaling-host");
    
    // Configure API key authorization: ApiKeyAuth
    ApiKeyAuth ApiKeyAuth = (ApiKeyAuth) defaultClient.getAuthentication("ApiKeyAuth");
    ApiKeyAuth.setApiKey("YOUR API KEY");
    // Uncomment the following line to set a prefix for the API key, e.g. "Token" (defaults to null)
    //ApiKeyAuth.setApiKeyPrefix("Token");

    // Configure HTTP bearer authorization: BearerAuth
    HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
    BearerAuth.setBearerToken("BEARER TOKEN");

    DeviceGroupsApi apiInstance = new DeviceGroupsApi(defaultClient);
    Long groupId = 56L; // Long | The numeric id of one of the caller's private device groups.
    UserDeviceGroupDevicesInput userDeviceGroupDevicesInput = new UserDeviceGroupDevicesInput(); // UserDeviceGroupDevicesInput | 
    try {
      UserDeviceGroup result = apiInstance.replaceMyDeviceGroupDevices(groupId, userDeviceGroupDevicesInput);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling DeviceGroupsApi#replaceMyDeviceGroupDevices");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters

| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **groupId** | **Long**| The numeric id of one of the caller&#39;s private device groups. | |
| **userDeviceGroupDevicesInput** | [**UserDeviceGroupDevicesInput**](UserDeviceGroupDevicesInput.md)|  | |

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
| **200** | The updated group. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="updateMyDeviceGroup"></a>
# **updateMyDeviceGroup**
> UserDeviceGroup updateMyDeviceGroup(groupId, userDeviceGroupPatch)

Update one of the caller&#39;s address-book groups

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.DeviceGroupsApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("https://your-signaling-host");
    
    // Configure API key authorization: ApiKeyAuth
    ApiKeyAuth ApiKeyAuth = (ApiKeyAuth) defaultClient.getAuthentication("ApiKeyAuth");
    ApiKeyAuth.setApiKey("YOUR API KEY");
    // Uncomment the following line to set a prefix for the API key, e.g. "Token" (defaults to null)
    //ApiKeyAuth.setApiKeyPrefix("Token");

    // Configure HTTP bearer authorization: BearerAuth
    HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
    BearerAuth.setBearerToken("BEARER TOKEN");

    DeviceGroupsApi apiInstance = new DeviceGroupsApi(defaultClient);
    Long groupId = 56L; // Long | The numeric id of one of the caller's private device groups.
    UserDeviceGroupPatch userDeviceGroupPatch = new UserDeviceGroupPatch(); // UserDeviceGroupPatch | 
    try {
      UserDeviceGroup result = apiInstance.updateMyDeviceGroup(groupId, userDeviceGroupPatch);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling DeviceGroupsApi#updateMyDeviceGroup");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters

| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **groupId** | **Long**| The numeric id of one of the caller&#39;s private device groups. | |
| **userDeviceGroupPatch** | [**UserDeviceGroupPatch**](UserDeviceGroupPatch.md)|  | |

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
| **200** | The updated group. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |
| **409** | The caller already has a group with that name. |  -  |

