# OpenAPI\Client\UpdateApi

All URIs are relative to https://your-signaling-host, except if the operation defines another base path.

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**checkForUpdate()**](UpdateApi.md#checkForUpdate) | **GET** /v1/update:check | Check the self-hosted update channel for a newer build |
| [**downloadUpdateArtifact()**](UpdateApi.md#downloadUpdateArtifact) | **GET** /v1/update-releases/{release_id}/artifact | Download an installer this server hosts itself |


## `checkForUpdate()`

```php
checkForUpdate($platform, $arch, $channel, $current_version): \OpenAPI\Client\Model\UpdateCheckResult
```

Check the self-hosted update channel for a newer build

The client-facing half of the self-hosted update channel. It is API-key gated but not user-authenticated: a controlled machine checks for updates before anyone logs in.  The server owns the index of published releases. The artifact lives at `download_url`: normally a host the operator controls, or this server itself when the installer was uploaded to it (then `download_url` is `GET /v1/update-releases/{release_id}/artifact` and `hosted_here` is true). The index is managed through the admin API (`POST /v1/admin/update-releases` and friends), which stays outside this business contract.  `current_version` is optional. Without it the newest release is returned and `update_available` is true whenever one exists; with it, `update_available` is true only when the published version is strictly newer, and `mandatory` mirrors the release flag once an update is available. `latest` is always present and is `null` when the channel has no release for that platform/arch.

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


$apiInstance = new OpenAPI\Client\Api\UpdateApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$platform = 'platform_example'; // string | Which operating system the caller is running.
$arch = 'arch_example'; // string | Which architecture the caller is running.
$channel = 'channel_example'; // string | Defaults to `stable`.
$current_version = 'current_version_example'; // string | The dotted numeric version the caller is running, e.g. `4.2.5.0`. A leading `v` is accepted.

try {
    $result = $apiInstance->checkForUpdate($platform, $arch, $channel, $current_version);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling UpdateApi->checkForUpdate: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **platform** | **string**| Which operating system the caller is running. | |
| **arch** | **string**| Which architecture the caller is running. | |
| **channel** | **string**| Defaults to &#x60;stable&#x60;. | [optional] |
| **current_version** | **string**| The dotted numeric version the caller is running, e.g. &#x60;4.2.5.0&#x60;. A leading &#x60;v&#x60; is accepted. | [optional] |

### Return type

[**\OpenAPI\Client\Model\UpdateCheckResult**](../Model/UpdateCheckResult.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `downloadUpdateArtifact()`

```php
downloadUpdateArtifact($release_id): \SplFileObject
```

Download an installer this server hosts itself

The other half of a re-hosted release. `GET /v1/update:check` returns the `download_url` of the newest build, and for a release whose bytes were uploaded to this server that URL is this route, under the same API key the client already used for the check.  API-key gated but not user-authenticated, like the check itself. The response supports range requests, so a transfer that drops can resume.  A release whose bytes live on an external static host is not served here: fetching its route is a `404`, and the client should use the `download_url` from the check response instead.

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


$apiInstance = new OpenAPI\Client\Api\UpdateApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$release_id = 56; // int | The `id` of the release from the check response.

try {
    $result = $apiInstance->downloadUpdateArtifact($release_id);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling UpdateApi->downloadUpdateArtifact: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **release_id** | **int**| The &#x60;id&#x60; of the release from the check response. | |

### Return type

**\SplFileObject**

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/octet-stream`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)
