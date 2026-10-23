# OpenAPI\Client\FileTransfersApi

All URIs are relative to https://your-signaling-host, except if the operation defines another base path.

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**createClipboardAudit()**](FileTransfersApi.md#createClipboardAudit) | **POST** /v1/me/clipboard-audits | Report a controller-observed clipboard sync for auditing |
| [**createFileTransferAudit()**](FileTransfersApi.md#createFileTransferAudit) | **POST** /v1/me/file-transfer-audits | Report a controller-observed file transfer for auditing |


## `createClipboardAudit()`

```php
createClipboardAudit($clipboard_audit_input)
```

Report a controller-observed clipboard sync for auditing

Clipboard bytes travel directly over the Remoting data channel and never pass through the signaling server. This endpoint lets an authenticated device owner report a clipboard sync after the fact so it appears in the audit trail.  The report carries **metadata only**: direction, MIME type and byte count. There is deliberately no field for the clipboard text. A clipboard is by definition whatever the user last copied - routinely a password, a recovery code or a private message - so a row that stored the text would turn the audit trail into a credential store every later reader of the database inherits. \"The operator copied 1.2 KB of text at 14:03\" answers the investigative question; the text answers nothing an investigator needs and costs everything if the database leaks.  Reports are idempotent by `transfer_id`, and the device must belong to the caller. This is an attributable controller report, not cryptographic proof from the host that a clipboard sync happened.

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


$apiInstance = new OpenAPI\Client\Api\FileTransfersApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$clipboard_audit_input = new \OpenAPI\Client\Model\ClipboardAuditInput(); // \OpenAPI\Client\Model\ClipboardAuditInput

try {
    $apiInstance->createClipboardAudit($clipboard_audit_input);
} catch (Exception $e) {
    echo 'Exception when calling FileTransfersApi->createClipboardAudit: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **clipboard_audit_input** | [**\OpenAPI\Client\Model\ClipboardAuditInput**](../Model/ClipboardAuditInput.md)|  | |

### Return type

void (empty response body)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `createFileTransferAudit()`

```php
createFileTransferAudit($file_transfer_audit_input)
```

Report a controller-observed file transfer for auditing

File bytes travel directly over the Remoting data channel and never pass through the signaling server. This endpoint lets an authenticated device owner report a completed or failed transfer after the fact so it appears in the audit trail.  Reports are idempotent by `(actor, direction, transfer_id)`: retrying the same event does not create another row. The device must belong to the caller. This is an attributable controller report, not cryptographic proof from the host that a file operation happened.

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


$apiInstance = new OpenAPI\Client\Api\FileTransfersApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$file_transfer_audit_input = new \OpenAPI\Client\Model\FileTransferAuditInput(); // \OpenAPI\Client\Model\FileTransferAuditInput

try {
    $apiInstance->createFileTransferAudit($file_transfer_audit_input);
} catch (Exception $e) {
    echo 'Exception when calling FileTransfersApi->createFileTransferAudit: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **file_transfer_audit_input** | [**\OpenAPI\Client\Model\FileTransferAuditInput**](../Model/FileTransferAuditInput.md)|  | |

### Return type

void (empty response body)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)
