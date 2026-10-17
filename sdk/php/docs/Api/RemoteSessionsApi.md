# OpenAPI\Client\RemoteSessionsApi

All URIs are relative to https://your-signaling-host, except if the operation defines another base path.

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**getRemoteSession()**](RemoteSessionsApi.md#getRemoteSession) | **GET** /v1/me/remote-sessions/{session_id} | One control session with the records that share its id |
| [**listDeviceRemoteSessions()**](RemoteSessionsApi.md#listDeviceRemoteSessions) | **GET** /v1/me/devices/{device_id}/sessions | One device&#39;s control sessions, each with its related records |
| [**listRemoteSessions()**](RemoteSessionsApi.md#listRemoteSessions) | **GET** /v1/me/remote-sessions | The caller&#39;s control sessions, open and finished |
| [**revokeDeviceSessions()**](RemoteSessionsApi.md#revokeDeviceSessions) | **POST** /v1/me/devices/{device_id}/sessions:revoke | End every live control session on one device |
| [**revokeRemoteSession()**](RemoteSessionsApi.md#revokeRemoteSession) | **POST** /v1/me/remote-sessions/{session_id}/revoke | End one of the caller&#39;s control sessions |


## `getRemoteSession()`

```php
getRemoteSession($session_id): \OpenAPI\Client\Model\RemoteSessionDetail
```

One control session with the records that share its id

What happened in one session: the session row itself, the recordings produced during it, and the device access-log entries that carry its `session_id` (in practice the `session_started` row written when the controller authenticated). The three tables always existed - this is the read that puts them back together.  Owner only. A session belonging to another account is reported as `404` rather than `403`, so a caller cannot probe which session ids exist. Both lists are capped at 100 entries and are empty arrays rather than absent when there is nothing to show.

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


$apiInstance = new OpenAPI\Client\Api\RemoteSessionsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$session_id = 'session_id_example'; // string | The signaling session id from the list.

try {
    $result = $apiInstance->getRemoteSession($session_id);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling RemoteSessionsApi->getRemoteSession: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **session_id** | **string**| The signaling session id from the list. | |

### Return type

[**\OpenAPI\Client\Model\RemoteSessionDetail**](../Model/RemoteSessionDetail.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `listDeviceRemoteSessions()`

```php
listDeviceRemoteSessions($device_id, $status, $cursor, $limit): \OpenAPI\Client\Model\RemoteSessionDetailList
```

One device's control sessions, each with its related records

The same history as `GET /v1/me/remote-sessions?device_id=`, but every entry arrives already joined with the records that share its session id - the shape a device page needs, so a client never has to know a session id (or make a second request per row) to reach the join.  `status` is `active`, `ended` or omitted/`all`; `cursor`/`limit` page it newest first like every other list here. The related arrays are batched across the page, so a page costs three queries rather than three per row, and they are present-but-empty rather than absent.  A device the caller does not own is `403` and an unknown device is `404`, matching `.../access-logs`. (The session detail read answers `404` for somebody else's session instead, because a session id is looked up rather than named by its owner.)

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


$apiInstance = new OpenAPI\Client\Api\RemoteSessionsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$device_id = 'device_id_example'; // string | The device id shown on the controlled machine.
$status = 'status_example'; // string | Defaults to `all`.
$cursor = 'cursor_example'; // string
$limit = 56; // int

try {
    $result = $apiInstance->listDeviceRemoteSessions($device_id, $status, $cursor, $limit);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling RemoteSessionsApi->listDeviceRemoteSessions: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **device_id** | **string**| The device id shown on the controlled machine. | |
| **status** | **string**| Defaults to &#x60;all&#x60;. | [optional] |
| **cursor** | **string**|  | [optional] |
| **limit** | **int**|  | [optional] |

### Return type

[**\OpenAPI\Client\Model\RemoteSessionDetailList**](../Model/RemoteSessionDetailList.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `listRemoteSessions()`

```php
listRemoteSessions($status, $device_id, $cursor, $limit): \OpenAPI\Client\Model\RemoteSessionList
```

The caller's control sessions, open and finished

Who is in the caller's machines right now, and who has been.  A remote session is one controller holding a signaling session against a device. It is opened when that controller's signaling socket completes authentication and closed when that socket ends, which is the whole lifetime this server can observe: the media path is peer to peer, so a session whose signaling channel is gone is over as far as the control plane is concerned.  `status=active` answers \"who is in right now\", `ended` is history, and the default lists both. `device_id` narrows to one machine. Cursor-paginated, newest first, `next_cursor` absent on the last page.

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


$apiInstance = new OpenAPI\Client\Api\RemoteSessionsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$status = 'status_example'; // string | Defaults to `all`.
$device_id = 'device_id_example'; // string
$cursor = 'cursor_example'; // string
$limit = 56; // int

try {
    $result = $apiInstance->listRemoteSessions($status, $device_id, $cursor, $limit);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling RemoteSessionsApi->listRemoteSessions: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **status** | **string**| Defaults to &#x60;all&#x60;. | [optional] |
| **device_id** | **string**|  | [optional] |
| **cursor** | **string**|  | [optional] |
| **limit** | **int**|  | [optional] |

### Return type

[**\OpenAPI\Client\Model\RemoteSessionList**](../Model/RemoteSessionList.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `revokeDeviceSessions()`

```php
revokeDeviceSessions($device_id): \OpenAPI\Client\Model\RemoteSessionBatchRevocation
```

End every live control session on one device

The owner's \"throw everyone out of this machine\": every live controller on the device is ended in one call, whichever server instance happens to hold each socket. Each is closed at its owning instance, and the call confirms the database row before reporting it - the same promise the single-session revoke makes.  A batch does not fail as a whole: one session that cannot be reached must not stop the others, so the response reports each session's outcome and counts. `truncated` says the device had more live controllers than one call handles (the cap is 200), which means the machine is not necessarily empty when the call returns. A machine with nothing live is a `200` with `revoked: 0`, not an error.  Ownership answers match the sibling routes: `404` for a device that does not exist, `403` for one that belongs to somebody else.

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


$apiInstance = new OpenAPI\Client\Api\RemoteSessionsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$device_id = 'device_id_example'; // string | The device id shown on the controlled machine.

try {
    $result = $apiInstance->revokeDeviceSessions($device_id);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling RemoteSessionsApi->revokeDeviceSessions: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **device_id** | **string**| The device id shown on the controlled machine. | |

### Return type

[**\OpenAPI\Client\Model\RemoteSessionBatchRevocation**](../Model/RemoteSessionBatchRevocation.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `revokeRemoteSession()`

```php
revokeRemoteSession($session_id): \OpenAPI\Client\Model\RemoteSessionRevocation
```

End one of the caller's control sessions

The owner's \"throw them out\". A remote session lives exactly as long as the controller's signaling socket, so ending the session means closing that socket; the session row is stamped `end_reason: revoked` and the owner's realtime stream carries `remote_session.ended` with that reason. Closing the socket also frees the device's controller slot, so a revoked session stops counting against the access policy's `max_sessions`.  Owner only, with the same concealment as the detail read: a session that belongs to another account is a `404`, not a `403`. The socket may be held by a different server instance - the row records which one admitted it - so the request is addressed at that instance and this call waits for the session to actually close: a `200` means it happened, not that a command was merely published. A session that is already over, or whose owning instance did not close it in time, is a `409`, because the one thing worse than refusing a revoke is reporting a success that did not happen.

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


$apiInstance = new OpenAPI\Client\Api\RemoteSessionsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$session_id = 'session_id_example'; // string | The signaling session id from the list.

try {
    $result = $apiInstance->revokeRemoteSession($session_id);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling RemoteSessionsApi->revokeRemoteSession: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **session_id** | **string**| The signaling session id from the list. | |

### Return type

[**\OpenAPI\Client\Model\RemoteSessionRevocation**](../Model/RemoteSessionRevocation.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)
