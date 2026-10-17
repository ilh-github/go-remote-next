# OpenAPI\Client\AuthApi

All URIs are relative to https://your-signaling-host, except if the operation defines another base path.

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**createSession()**](AuthApi.md#createSession) | **POST** /v1/auth/sessions | Exchange a username/password for a session |


## `createSession()`

```php
createSession($auth_session_request): \OpenAPI\Client\Model\AuthSessionResponse
```

Exchange a username/password for a session

### Example

```php
<?php
require_once(__DIR__ . '/vendor/autoload.php');


// Configure API key authorization: ApiKeyAuth
$config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setApiKey('X-API-Key', 'YOUR_API_KEY');
// Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
// $config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setApiKeyPrefix('X-API-Key', 'Bearer');


$apiInstance = new OpenAPI\Client\Api\AuthApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$auth_session_request = new \OpenAPI\Client\Model\AuthSessionRequest(); // \OpenAPI\Client\Model\AuthSessionRequest

try {
    $result = $apiInstance->createSession($auth_session_request);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling AuthApi->createSession: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **auth_session_request** | [**\OpenAPI\Client\Model\AuthSessionRequest**](../Model/AuthSessionRequest.md)|  | |

### Return type

[**\OpenAPI\Client\Model\AuthSessionResponse**](../Model/AuthSessionResponse.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)
