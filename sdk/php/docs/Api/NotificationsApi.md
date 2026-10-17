# OpenAPI\Client\NotificationsApi

All URIs are relative to https://your-signaling-host, except if the operation defines another base path.

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**getOfflineAlert()**](NotificationsApi.md#getOfflineAlert) | **GET** /v1/me/devices/{device_id}/offline-alert | The owner&#39;s offline-alert rule for a device |
| [**listNotificationChannels()**](NotificationsApi.md#listNotificationChannels) | **GET** /v1/me/notification-channels | The user&#39;s notification channels |
| [**putOfflineAlert()**](NotificationsApi.md#putOfflineAlert) | **PUT** /v1/me/devices/{device_id}/offline-alert | Create or replace the owner&#39;s offline-alert rule |


## `getOfflineAlert()`

```php
getOfflineAlert($device_id): \OpenAPI\Client\Model\OfflineAlertRule
```

The owner's offline-alert rule for a device

Returns the documented defaults (`enabled: false`, `after_minutes: 10`) when the owner never configured a rule, so a client can render the form without a second \"does it exist\" call.

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


$apiInstance = new OpenAPI\Client\Api\NotificationsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$device_id = 'device_id_example'; // string | The device id shown on the controlled machine.

try {
    $result = $apiInstance->getOfflineAlert($device_id);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling NotificationsApi->getOfflineAlert: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **device_id** | **string**| The device id shown on the controlled machine. | |

### Return type

[**\OpenAPI\Client\Model\OfflineAlertRule**](../Model/OfflineAlertRule.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `listNotificationChannels()`

```php
listNotificationChannels(): \OpenAPI\Client\Model\NotificationChannelList
```

The user's notification channels

`supported_events` is the authoritative list of event types a channel may subscribe to. The signing secret is never returned; deliveries are signed with `signature_header` using `signature_scheme`.

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


$apiInstance = new OpenAPI\Client\Api\NotificationsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);

try {
    $result = $apiInstance->listNotificationChannels();
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling NotificationsApi->listNotificationChannels: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**\OpenAPI\Client\Model\NotificationChannelList**](../Model/NotificationChannelList.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `putOfflineAlert()`

```php
putOfflineAlert($device_id, $offline_alert_input): \OpenAPI\Client\Model\OfflineAlertRule
```

Create or replace the owner's offline-alert rule

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


$apiInstance = new OpenAPI\Client\Api\NotificationsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$device_id = 'device_id_example'; // string | The device id shown on the controlled machine.
$offline_alert_input = new \OpenAPI\Client\Model\OfflineAlertInput(); // \OpenAPI\Client\Model\OfflineAlertInput

try {
    $result = $apiInstance->putOfflineAlert($device_id, $offline_alert_input);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling NotificationsApi->putOfflineAlert: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **device_id** | **string**| The device id shown on the controlled machine. | |
| **offline_alert_input** | [**\OpenAPI\Client\Model\OfflineAlertInput**](../Model/OfflineAlertInput.md)|  | |

### Return type

[**\OpenAPI\Client\Model\OfflineAlertRule**](../Model/OfflineAlertRule.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)
