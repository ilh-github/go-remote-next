# OpenAPI\Client\DeviceGroupsApi

All URIs are relative to https://your-signaling-host, except if the operation defines another base path.

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**createMyDeviceGroup()**](DeviceGroupsApi.md#createMyDeviceGroup) | **POST** /v1/me/device-groups | Create a private address-book group |
| [**deleteMyDeviceGroup()**](DeviceGroupsApi.md#deleteMyDeviceGroup) | **DELETE** /v1/me/device-groups/{group_id} | Delete one of the caller&#39;s address-book groups |
| [**listMyDeviceGroups()**](DeviceGroupsApi.md#listMyDeviceGroups) | **GET** /v1/me/device-groups | The caller&#39;s private address-book groups |
| [**replaceMyDeviceGroupDevices()**](DeviceGroupsApi.md#replaceMyDeviceGroupDevices) | **PUT** /v1/me/device-groups/{group_id}/devices | Replace the devices in one of the caller&#39;s groups |
| [**updateMyDeviceGroup()**](DeviceGroupsApi.md#updateMyDeviceGroup) | **PATCH** /v1/me/device-groups/{group_id} | Update one of the caller&#39;s address-book groups |


## `createMyDeviceGroup()`

```php
createMyDeviceGroup($user_device_group_input): \OpenAPI\Client\Model\UserDeviceGroup
```

Create a private address-book group

### Example

```php
<?php
require_once(__DIR__ . '/vendor/autoload.php');


// Configure API key authorization: ApiKeyAuth
$config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setApiKey('X-API-Key', 'YOUR_API_KEY');
// Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
// $config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setApiKeyPrefix('X-API-Key', 'Bearer');

// Configure Bearer (opaque) authorization: BearerAuth
$config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setAccessToken('YOUR_ACCESS_TOKEN');


$apiInstance = new OpenAPI\Client\Api\DeviceGroupsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$user_device_group_input = new \OpenAPI\Client\Model\UserDeviceGroupInput(); // \OpenAPI\Client\Model\UserDeviceGroupInput

try {
    $result = $apiInstance->createMyDeviceGroup($user_device_group_input);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling DeviceGroupsApi->createMyDeviceGroup: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **user_device_group_input** | [**\OpenAPI\Client\Model\UserDeviceGroupInput**](../Model/UserDeviceGroupInput.md)|  | |

### Return type

[**\OpenAPI\Client\Model\UserDeviceGroup**](../Model/UserDeviceGroup.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `deleteMyDeviceGroup()`

```php
deleteMyDeviceGroup($group_id): \OpenAPI\Client\Model\DeleteMyDeviceGroup200Response
```

Delete one of the caller's address-book groups

### Example

```php
<?php
require_once(__DIR__ . '/vendor/autoload.php');


// Configure API key authorization: ApiKeyAuth
$config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setApiKey('X-API-Key', 'YOUR_API_KEY');
// Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
// $config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setApiKeyPrefix('X-API-Key', 'Bearer');

// Configure Bearer (opaque) authorization: BearerAuth
$config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setAccessToken('YOUR_ACCESS_TOKEN');


$apiInstance = new OpenAPI\Client\Api\DeviceGroupsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$group_id = 56; // int | The numeric id of one of the caller's private device groups.

try {
    $result = $apiInstance->deleteMyDeviceGroup($group_id);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling DeviceGroupsApi->deleteMyDeviceGroup: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **group_id** | **int**| The numeric id of one of the caller&#39;s private device groups. | |

### Return type

[**\OpenAPI\Client\Model\DeleteMyDeviceGroup200Response**](../Model/DeleteMyDeviceGroup200Response.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `listMyDeviceGroups()`

```php
listMyDeviceGroups(): \OpenAPI\Client\Model\UserDeviceGroupList
```

The caller's private address-book groups

Owner-scoped groups are separate from the administrator-wide device inventory under `/v1/admin/groups`. Each returned group carries its current `device_ids`, so a client can render filters and membership without issuing one request per group.

### Example

```php
<?php
require_once(__DIR__ . '/vendor/autoload.php');


// Configure API key authorization: ApiKeyAuth
$config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setApiKey('X-API-Key', 'YOUR_API_KEY');
// Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
// $config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setApiKeyPrefix('X-API-Key', 'Bearer');

// Configure Bearer (opaque) authorization: BearerAuth
$config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setAccessToken('YOUR_ACCESS_TOKEN');


$apiInstance = new OpenAPI\Client\Api\DeviceGroupsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);

try {
    $result = $apiInstance->listMyDeviceGroups();
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling DeviceGroupsApi->listMyDeviceGroups: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**\OpenAPI\Client\Model\UserDeviceGroupList**](../Model/UserDeviceGroupList.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `replaceMyDeviceGroupDevices()`

```php
replaceMyDeviceGroupDevices($group_id, $user_device_group_devices_input): \OpenAPI\Client\Model\UserDeviceGroup
```

Replace the devices in one of the caller's groups

Replaces the whole membership atomically. Every device must currently belong to the caller; an unknown or foreign device is rejected with `403 DEVICE_NOT_OWNED`. Sending an empty `device_ids` array clears the group.

### Example

```php
<?php
require_once(__DIR__ . '/vendor/autoload.php');


// Configure API key authorization: ApiKeyAuth
$config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setApiKey('X-API-Key', 'YOUR_API_KEY');
// Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
// $config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setApiKeyPrefix('X-API-Key', 'Bearer');

// Configure Bearer (opaque) authorization: BearerAuth
$config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setAccessToken('YOUR_ACCESS_TOKEN');


$apiInstance = new OpenAPI\Client\Api\DeviceGroupsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$group_id = 56; // int | The numeric id of one of the caller's private device groups.
$user_device_group_devices_input = new \OpenAPI\Client\Model\UserDeviceGroupDevicesInput(); // \OpenAPI\Client\Model\UserDeviceGroupDevicesInput

try {
    $result = $apiInstance->replaceMyDeviceGroupDevices($group_id, $user_device_group_devices_input);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling DeviceGroupsApi->replaceMyDeviceGroupDevices: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **group_id** | **int**| The numeric id of one of the caller&#39;s private device groups. | |
| **user_device_group_devices_input** | [**\OpenAPI\Client\Model\UserDeviceGroupDevicesInput**](../Model/UserDeviceGroupDevicesInput.md)|  | |

### Return type

[**\OpenAPI\Client\Model\UserDeviceGroup**](../Model/UserDeviceGroup.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `updateMyDeviceGroup()`

```php
updateMyDeviceGroup($group_id, $user_device_group_patch): \OpenAPI\Client\Model\UserDeviceGroup
```

Update one of the caller's address-book groups

### Example

```php
<?php
require_once(__DIR__ . '/vendor/autoload.php');


// Configure API key authorization: ApiKeyAuth
$config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setApiKey('X-API-Key', 'YOUR_API_KEY');
// Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
// $config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setApiKeyPrefix('X-API-Key', 'Bearer');

// Configure Bearer (opaque) authorization: BearerAuth
$config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setAccessToken('YOUR_ACCESS_TOKEN');


$apiInstance = new OpenAPI\Client\Api\DeviceGroupsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$group_id = 56; // int | The numeric id of one of the caller's private device groups.
$user_device_group_patch = new \OpenAPI\Client\Model\UserDeviceGroupPatch(); // \OpenAPI\Client\Model\UserDeviceGroupPatch

try {
    $result = $apiInstance->updateMyDeviceGroup($group_id, $user_device_group_patch);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling DeviceGroupsApi->updateMyDeviceGroup: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **group_id** | **int**| The numeric id of one of the caller&#39;s private device groups. | |
| **user_device_group_patch** | [**\OpenAPI\Client\Model\UserDeviceGroupPatch**](../Model/UserDeviceGroupPatch.md)|  | |

### Return type

[**\OpenAPI\Client\Model\UserDeviceGroup**](../Model/UserDeviceGroup.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)
