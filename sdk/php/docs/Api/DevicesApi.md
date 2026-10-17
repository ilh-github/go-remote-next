# OpenAPI\Client\DevicesApi

All URIs are relative to https://your-signaling-host, except if the operation defines another base path.

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**approveAccessApproval()**](DevicesApi.md#approveAccessApproval) | **POST** /v1/me/devices/{device_id}/access-approvals/{approval_id}/approve | Approve a pending access request |
| [**banDeviceAccessSource()**](DevicesApi.md#banDeviceAccessSource) | **POST** /v1/me/devices/{device_id}/access-bans | Permanently deny one source address |
| [**claimAccessApproval()**](DevicesApi.md#claimAccessApproval) | **POST** /v1/devices/{device_id}/access-approvals:claim | Exchange an approved request for a signal token |
| [**createAccessGrant()**](DevicesApi.md#createAccessGrant) | **POST** /v1/me/devices/{device_id}/access-grants | Issue a time-boxed guest access code |
| [**denyAccessApproval()**](DevicesApi.md#denyAccessApproval) | **POST** /v1/me/devices/{device_id}/access-approvals/{approval_id}/deny | Deny a pending access request |
| [**getAccessApprovalSummary()**](DevicesApi.md#getAccessApprovalSummary) | **GET** /v1/me/devices/{device_id}/access-approvals:summary | Aggregated view of a device&#39;s approval queue |
| [**getAccessPolicy()**](DevicesApi.md#getAccessPolicy) | **GET** /v1/me/devices/{device_id}/access-policy | The device&#39;s access policy |
| [**listAccessApprovals()**](DevicesApi.md#listAccessApprovals) | **GET** /v1/me/devices/{device_id}/access-approvals | Second-factor approvals awaiting (or already decided by) the owner |
| [**listAccessGrants()**](DevicesApi.md#listAccessGrants) | **GET** /v1/me/devices/{device_id}/access-grants | Guest access codes issued for a device |
| [**listMyDevices()**](DevicesApi.md#listMyDevices) | **GET** /v1/me/devices | Devices owned by the authenticated user |
| [**putAccessPolicy()**](DevicesApi.md#putAccessPolicy) | **PUT** /v1/me/devices/{device_id}/access-policy | Create or replace the device&#39;s access policy |
| [**reportHostMetrics()**](DevicesApi.md#reportHostMetrics) | **POST** /v1/devices/{device_id}/metrics | Report the host&#39;s latest resource sample |
| [**revokeAccessGrant()**](DevicesApi.md#revokeAccessGrant) | **DELETE** /v1/me/devices/{device_id}/access-grants/{grant_id} | Revoke a guest access code |
| [**unbanDeviceAccessSource()**](DevicesApi.md#unbanDeviceAccessSource) | **POST** /v1/me/devices/{device_id}/access-bans:remove | Remove one permanent source ban |
| [**verifyAccessCode()**](DevicesApi.md#verifyAccessCode) | **POST** /v1/devices/{device_id}/access-code:verify | Verify a device access code and mint a signal token |


## `approveAccessApproval()`

```php
approveAccessApproval($device_id, $approval_id, $approve_access_approval_request): \OpenAPI\Client\Model\AccessApproval
```

Approve a pending access request

Owner-only. The body is optional; a `reason` is stored on the row and shown to the requester. A request that was already decided or has expired returns `409` with the current status instead of pretending the decision landed. The conformance smoke pins the `404` shape with an unknown id; the happy path needs a real pending approval (an online device plus `approval_mode`), so it is covered by the end-to-end run in `docs/dev/自托管端到端联调记录.md` rather than the smoke.

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


$apiInstance = new OpenAPI\Client\Api\DevicesApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$device_id = 'device_id_example'; // string | The device id shown on the controlled machine.
$approval_id = 56; // int | The numeric id from an approval row.
$approve_access_approval_request = new \OpenAPI\Client\Model\ApproveAccessApprovalRequest(); // \OpenAPI\Client\Model\ApproveAccessApprovalRequest

try {
    $result = $apiInstance->approveAccessApproval($device_id, $approval_id, $approve_access_approval_request);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling DevicesApi->approveAccessApproval: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **device_id** | **string**| The device id shown on the controlled machine. | |
| **approval_id** | **int**| The numeric id from an approval row. | |
| **approve_access_approval_request** | [**\OpenAPI\Client\Model\ApproveAccessApprovalRequest**](../Model/ApproveAccessApprovalRequest.md)|  | [optional] |

### Return type

[**\OpenAPI\Client\Model\AccessApproval**](../Model/AccessApproval.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `banDeviceAccessSource()`

```php
banDeviceAccessSource($device_id, $device_access_ban_request): \OpenAPI\Client\Model\DeviceAccessPolicy
```

Permanently deny one source address

Adds an exact IP or CIDR block to the device's persistent deny list and returns the complete updated policy. A deny match always wins over the allow list, time windows and a valid access code. Repeating the same source is a no-op; remove one source with `POST /v1/me/devices/{device_id}/access-bans:remove`.

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


$apiInstance = new OpenAPI\Client\Api\DevicesApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$device_id = 'device_id_example'; // string | The device id shown on the controlled machine.
$device_access_ban_request = new \OpenAPI\Client\Model\DeviceAccessBanRequest(); // \OpenAPI\Client\Model\DeviceAccessBanRequest

try {
    $result = $apiInstance->banDeviceAccessSource($device_id, $device_access_ban_request);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling DevicesApi->banDeviceAccessSource: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **device_id** | **string**| The device id shown on the controlled machine. | |
| **device_access_ban_request** | [**\OpenAPI\Client\Model\DeviceAccessBanRequest**](../Model/DeviceAccessBanRequest.md)|  | |

### Return type

[**\OpenAPI\Client\Model\DeviceAccessPolicy**](../Model/DeviceAccessPolicy.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `claimAccessApproval()`

```php
claimAccessApproval($device_id, $approval_claim_request): \OpenAPI\Client\Model\AccessCodeVerifyResponse
```

Exchange an approved request for a signal token

The requesting controller calls this with the `approval_id` and `approval_secret` it received from `access-code:verify`. It is API-key gated rather than user-authenticated because the caller has no account. `202` means the owner has not decided yet - keep polling with the same id and secret; `200` returns the signal token; `403` means denied or expired; `409` means the approval was already spent or the host went offline before the exchange.

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


$apiInstance = new OpenAPI\Client\Api\DevicesApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$device_id = 'device_id_example'; // string | The device id shown on the controlled machine.
$approval_claim_request = new \OpenAPI\Client\Model\ApprovalClaimRequest(); // \OpenAPI\Client\Model\ApprovalClaimRequest

try {
    $result = $apiInstance->claimAccessApproval($device_id, $approval_claim_request);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling DevicesApi->claimAccessApproval: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **device_id** | **string**| The device id shown on the controlled machine. | |
| **approval_claim_request** | [**\OpenAPI\Client\Model\ApprovalClaimRequest**](../Model/ApprovalClaimRequest.md)|  | |

### Return type

[**\OpenAPI\Client\Model\AccessCodeVerifyResponse**](../Model/AccessCodeVerifyResponse.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `createAccessGrant()`

```php
createAccessGrant($device_id, $access_grant_input): \OpenAPI\Client\Model\AccessGrant
```

Issue a time-boxed guest access code

The plaintext code is returned **once**, in the creation response; a grant is limited by expiry and uses, may be bound to a client id and/or source addresses, and is revoked with the DELETE below.

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


$apiInstance = new OpenAPI\Client\Api\DevicesApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$device_id = 'device_id_example'; // string | The device id shown on the controlled machine.
$access_grant_input = new \OpenAPI\Client\Model\AccessGrantInput(); // \OpenAPI\Client\Model\AccessGrantInput

try {
    $result = $apiInstance->createAccessGrant($device_id, $access_grant_input);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling DevicesApi->createAccessGrant: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **device_id** | **string**| The device id shown on the controlled machine. | |
| **access_grant_input** | [**\OpenAPI\Client\Model\AccessGrantInput**](../Model/AccessGrantInput.md)|  | |

### Return type

[**\OpenAPI\Client\Model\AccessGrant**](../Model/AccessGrant.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `denyAccessApproval()`

```php
denyAccessApproval($device_id, $approval_id, $approve_access_approval_request): \OpenAPI\Client\Model\AccessApproval
```

Deny a pending access request

Owner-only, same shape as `approve`. Denying also arms the per-source denial cooldown (`APPROVAL_DENIAL_COOLDOWN_SECONDS`), so the same client cannot immediately re-request.

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


$apiInstance = new OpenAPI\Client\Api\DevicesApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$device_id = 'device_id_example'; // string | The device id shown on the controlled machine.
$approval_id = 56; // int | The numeric id from an approval row.
$approve_access_approval_request = new \OpenAPI\Client\Model\ApproveAccessApprovalRequest(); // \OpenAPI\Client\Model\ApproveAccessApprovalRequest

try {
    $result = $apiInstance->denyAccessApproval($device_id, $approval_id, $approve_access_approval_request);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling DevicesApi->denyAccessApproval: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **device_id** | **string**| The device id shown on the controlled machine. | |
| **approval_id** | **int**| The numeric id from an approval row. | |
| **approve_access_approval_request** | [**\OpenAPI\Client\Model\ApproveAccessApprovalRequest**](../Model/ApproveAccessApprovalRequest.md)|  | [optional] |

### Return type

[**\OpenAPI\Client\Model\AccessApproval**](../Model/AccessApproval.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `getAccessApprovalSummary()`

```php
getAccessApprovalSummary($device_id, $since, $limit): \OpenAPI\Client\Model\ApprovalSummary
```

Aggregated view of a device's approval queue

The row list shows requests one at a time; this is what makes \"one address has been refused six times this afternoon\" legible. The window is keyed on `created_at` for every count.

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


$apiInstance = new OpenAPI\Client\Api\DevicesApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$device_id = 'device_id_example'; // string | The device id shown on the controlled machine.
$since = new \DateTime('2013-10-20T19:20:30+01:00'); // \DateTime | RFC3339 start of the window; defaults to the last 24 hours.
$limit = 56; // int | How many refused sources to list (default 5, capped at 10).

try {
    $result = $apiInstance->getAccessApprovalSummary($device_id, $since, $limit);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling DevicesApi->getAccessApprovalSummary: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **device_id** | **string**| The device id shown on the controlled machine. | |
| **since** | **\DateTime**| RFC3339 start of the window; defaults to the last 24 hours. | [optional] |
| **limit** | **int**| How many refused sources to list (default 5, capped at 10). | [optional] |

### Return type

[**\OpenAPI\Client\Model\ApprovalSummary**](../Model/ApprovalSummary.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `getAccessPolicy()`

```php
getAccessPolicy($device_id): \OpenAPI\Client\Model\DeviceAccessPolicy
```

The device's access policy

Returns the documented \"no policy\" defaults when the owner never saved one, so a client can render the form without a second existence check.

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


$apiInstance = new OpenAPI\Client\Api\DevicesApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$device_id = 'device_id_example'; // string | The device id shown on the controlled machine.

try {
    $result = $apiInstance->getAccessPolicy($device_id);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling DevicesApi->getAccessPolicy: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **device_id** | **string**| The device id shown on the controlled machine. | |

### Return type

[**\OpenAPI\Client\Model\DeviceAccessPolicy**](../Model/DeviceAccessPolicy.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `listAccessApprovals()`

```php
listAccessApprovals($device_id, $status, $limit): \OpenAPI\Client\Model\ListAccessApprovals200Response
```

Second-factor approvals awaiting (or already decided by) the owner

`status` is a comma-separated subset of `pending, approved, denied, expired, consumed` and defaults to `pending`. Approval is created by the device-side verify route when the policy asks for it; approving or denying happens on the sub-resource routes documented in `docs/user-api-docs.md` §5.3.5 (not yet part of this contract).

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


$apiInstance = new OpenAPI\Client\Api\DevicesApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$device_id = 'device_id_example'; // string | The device id shown on the controlled machine.
$status = 'status_example'; // string
$limit = 56; // int

try {
    $result = $apiInstance->listAccessApprovals($device_id, $status, $limit);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling DevicesApi->listAccessApprovals: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **device_id** | **string**| The device id shown on the controlled machine. | |
| **status** | **string**|  | [optional] |
| **limit** | **int**|  | [optional] |

### Return type

[**\OpenAPI\Client\Model\ListAccessApprovals200Response**](../Model/ListAccessApprovals200Response.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `listAccessGrants()`

```php
listAccessGrants($device_id): \OpenAPI\Client\Model\ListAccessGrants200Response
```

Guest access codes issued for a device

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


$apiInstance = new OpenAPI\Client\Api\DevicesApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$device_id = 'device_id_example'; // string | The device id shown on the controlled machine.

try {
    $result = $apiInstance->listAccessGrants($device_id);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling DevicesApi->listAccessGrants: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **device_id** | **string**| The device id shown on the controlled machine. | |

### Return type

[**\OpenAPI\Client\Model\ListAccessGrants200Response**](../Model/ListAccessGrants200Response.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `listMyDevices()`

```php
listMyDevices(): \OpenAPI\Client\Model\ListMyDevices200Response
```

Devices owned by the authenticated user

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


$apiInstance = new OpenAPI\Client\Api\DevicesApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);

try {
    $result = $apiInstance->listMyDevices();
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling DevicesApi->listMyDevices: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**\OpenAPI\Client\Model\ListMyDevices200Response**](../Model/ListMyDevices200Response.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `putAccessPolicy()`

```php
putAccessPolicy($device_id, $device_access_policy_input): \OpenAPI\Client\Model\DeviceAccessPolicy
```

Create or replace the device's access policy

A **full replace**: every field not sent is reset to its zero value. The policy is enforced on the device-side verify route before the access code is compared, so a blocked client never gets to guess codes.

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


$apiInstance = new OpenAPI\Client\Api\DevicesApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$device_id = 'device_id_example'; // string | The device id shown on the controlled machine.
$device_access_policy_input = new \OpenAPI\Client\Model\DeviceAccessPolicyInput(); // \OpenAPI\Client\Model\DeviceAccessPolicyInput

try {
    $result = $apiInstance->putAccessPolicy($device_id, $device_access_policy_input);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling DevicesApi->putAccessPolicy: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **device_id** | **string**| The device id shown on the controlled machine. | |
| **device_access_policy_input** | [**\OpenAPI\Client\Model\DeviceAccessPolicyInput**](../Model/DeviceAccessPolicyInput.md)|  | |

### Return type

[**\OpenAPI\Client\Model\DeviceAccessPolicy**](../Model/DeviceAccessPolicy.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `reportHostMetrics()`

```php
reportHostMetrics($device_id, $host_metric_report): \OpenAPI\Client\Model\ReportHostMetrics200Response
```

Report the host's latest resource sample

Host-only endpoint authenticated by both the deployment API key and the device secret. Values are percentages in the inclusive range 0..100. Repeating a report replaces the previous sample and advances the owner's metric-alert episode state.

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


$apiInstance = new OpenAPI\Client\Api\DevicesApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$device_id = 'device_id_example'; // string | The device id shown on the controlled machine.
$host_metric_report = new \OpenAPI\Client\Model\HostMetricReport(); // \OpenAPI\Client\Model\HostMetricReport

try {
    $result = $apiInstance->reportHostMetrics($device_id, $host_metric_report);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling DevicesApi->reportHostMetrics: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **device_id** | **string**| The device id shown on the controlled machine. | |
| **host_metric_report** | [**\OpenAPI\Client\Model\HostMetricReport**](../Model/HostMetricReport.md)|  | |

### Return type

[**\OpenAPI\Client\Model\ReportHostMetrics200Response**](../Model/ReportHostMetrics200Response.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `revokeAccessGrant()`

```php
revokeAccessGrant($device_id, $grant_id): \OpenAPI\Client\Model\RevokeAccessGrant200Response
```

Revoke a guest access code

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


$apiInstance = new OpenAPI\Client\Api\DevicesApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$device_id = 'device_id_example'; // string | The device id shown on the controlled machine.
$grant_id = 56; // int

try {
    $result = $apiInstance->revokeAccessGrant($device_id, $grant_id);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling DevicesApi->revokeAccessGrant: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **device_id** | **string**| The device id shown on the controlled machine. | |
| **grant_id** | **int**|  | |

### Return type

[**\OpenAPI\Client\Model\RevokeAccessGrant200Response**](../Model/RevokeAccessGrant200Response.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `unbanDeviceAccessSource()`

```php
unbanDeviceAccessSource($device_id, $device_access_ban_request): \OpenAPI\Client\Model\DeviceAccessPolicy
```

Remove one permanent source ban

Removes the exact IP or CIDR entry from the device's persistent deny list and returns the complete updated policy. A source that is already absent, including on a device with no stored policy, is a successful no-op.

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


$apiInstance = new OpenAPI\Client\Api\DevicesApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$device_id = 'device_id_example'; // string | The device id shown on the controlled machine.
$device_access_ban_request = new \OpenAPI\Client\Model\DeviceAccessBanRequest(); // \OpenAPI\Client\Model\DeviceAccessBanRequest

try {
    $result = $apiInstance->unbanDeviceAccessSource($device_id, $device_access_ban_request);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling DevicesApi->unbanDeviceAccessSource: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **device_id** | **string**| The device id shown on the controlled machine. | |
| **device_access_ban_request** | [**\OpenAPI\Client\Model\DeviceAccessBanRequest**](../Model/DeviceAccessBanRequest.md)|  | |

### Return type

[**\OpenAPI\Client\Model\DeviceAccessPolicy**](../Model/DeviceAccessPolicy.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `verifyAccessCode()`

```php
verifyAccessCode($device_id, $access_code_verify_request): \OpenAPI\Client\Model\AccessCodeVerifyResponse
```

Verify a device access code and mint a signal token

The controller-side entry point. On success it returns a signal token for the relay. When the device policy wants the owner to confirm the session it returns `202` with a pending approval instead, and the caller polls `/v1/devices/{device_id}/access-approvals:claim`. A rejected code is a `403` problem with `code: INVALID_CODE`.  This route is reachable without a user token (the controlling client is not logged in) but still requires the API key or an allowed Origin.

### Example

```php
<?php
require_once(__DIR__ . '/vendor/autoload.php');


// Configure API key authorization: ApiKeyAuth
$config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setApiKey('X-API-Key', 'YOUR_API_KEY');
// Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
// $config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setApiKeyPrefix('X-API-Key', 'Bearer');


$apiInstance = new OpenAPI\Client\Api\DevicesApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$device_id = 'device_id_example'; // string | The device id shown on the controlled machine.
$access_code_verify_request = new \OpenAPI\Client\Model\AccessCodeVerifyRequest(); // \OpenAPI\Client\Model\AccessCodeVerifyRequest

try {
    $result = $apiInstance->verifyAccessCode($device_id, $access_code_verify_request);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling DevicesApi->verifyAccessCode: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **device_id** | **string**| The device id shown on the controlled machine. | |
| **access_code_verify_request** | [**\OpenAPI\Client\Model\AccessCodeVerifyRequest**](../Model/AccessCodeVerifyRequest.md)|  | |

### Return type

[**\OpenAPI\Client\Model\AccessCodeVerifyResponse**](../Model/AccessCodeVerifyResponse.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)
