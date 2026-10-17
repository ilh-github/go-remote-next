# OpenAPI\Client\RecordingsApi

All URIs are relative to https://your-signaling-host, except if the operation defines another base path.

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**appendRecordingArtifact()**](RecordingsApi.md#appendRecordingArtifact) | **PATCH** /v1/me/recordings/{recording_id}/artifact | Append one chunk of a resumable recording upload |
| [**createRecordingArtifactTicket()**](RecordingsApi.md#createRecordingArtifactTicket) | **POST** /v1/me/recordings/{recording_id}/artifact-ticket | Mint a short-lived playback URL for a recording |
| [**createRecordingThumbnailTicket()**](RecordingsApi.md#createRecordingThumbnailTicket) | **POST** /v1/me/recordings/{recording_id}/thumbnail-ticket | Mint a short-lived preview URL for a recording |
| [**createSessionRecording()**](RecordingsApi.md#createSessionRecording) | **POST** /v1/me/recordings | Index a new recording |
| [**deleteSessionRecording()**](RecordingsApi.md#deleteSessionRecording) | **DELETE** /v1/me/recordings/{recording_id} | Delete a recording and its stored media |
| [**downloadRecordingArtifact()**](RecordingsApi.md#downloadRecordingArtifact) | **GET** /v1/me/recordings/{recording_id}/artifact | Download a recording&#39;s media bytes |
| [**downloadRecordingArtifactByTicket()**](RecordingsApi.md#downloadRecordingArtifactByTicket) | **GET** /v1/recordings/{recording_id}/artifact | Fetch recording media with a playback ticket |
| [**downloadRecordingThumbnail()**](RecordingsApi.md#downloadRecordingThumbnail) | **GET** /v1/me/recordings/{recording_id}/thumbnail | Download a recording&#39;s preview image |
| [**downloadRecordingThumbnailByTicket()**](RecordingsApi.md#downloadRecordingThumbnailByTicket) | **GET** /v1/recordings/{recording_id}/thumbnail | Fetch a recording preview with a playback ticket |
| [**getSessionRecording()**](RecordingsApi.md#getSessionRecording) | **GET** /v1/me/recordings/{recording_id} | One recording the caller owns |
| [**listSessionRecordings()**](RecordingsApi.md#listSessionRecordings) | **GET** /v1/me/recordings | The caller&#39;s session-recording index |
| [**updateSessionRecording()**](RecordingsApi.md#updateSessionRecording) | **PATCH** /v1/me/recordings/{recording_id} | Update a recording&#39;s lifecycle metadata |
| [**uploadRecordingArtifact()**](RecordingsApi.md#uploadRecordingArtifact) | **POST** /v1/me/recordings/{recording_id}/artifact | Upload a recording&#39;s media bytes |
| [**uploadRecordingThumbnail()**](RecordingsApi.md#uploadRecordingThumbnail) | **POST** /v1/me/recordings/{recording_id}/thumbnail | Upload a recording&#39;s preview image |


## `appendRecordingArtifact()`

```php
appendRecordingArtifact($ext, $upload_offset, $recording_id, $upload_length, $upload_complete, $upload_probe, $body): \OpenAPI\Client\Model\SessionRecording
```

Append one chunk of a resumable recording upload

The resumable half of the upload contract, for a capture that does not fit in one request body or a connection that drops halfway. The client sends chunks; the server answers with the offset it actually holds in the `Upload-Offset` response header, so a retry resumes from that number instead of starting over.  Three headers drive it. `Upload-Offset` (required) is where the chunk starts and must equal the bytes already staged; otherwise the answer is `409` with the authoritative offset in the same header and the client resyncs. `Upload-Length` (optional, `*` while unknown) declares the total size, which lets the server finish by itself when the last byte lands and lets it refuse a chunk that would overshoot. `Upload-Complete: 1` says this chunk ends the upload, which is how a client finishes without ever declaring a total.  A status probe asks where the upload stands without staging bytes. Send `Upload-Probe: 1` with a valid `Upload-Offset` and no body; the answer is `202` with the current offset, so a client that crashed can re-learn where it stopped without a second endpoint. An empty body with a valid offset does the same thing for raw-HTTP callers, but a generated client cannot send a zero-length body, which is why the explicit header exists.  The staging file (`<artifact-path>.part`) is subject to the same size limit as the single-shot upload, an empty artifact is refused on every path, and the recording lifecycle deletes a staging file that has been untouched for six hours.

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


$apiInstance = new OpenAPI\Client\Api\RecordingsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$ext = 'ext_example'; // string | The artifact extension, allow-listed exactly like the raw-body POST shape.
$upload_offset = 56; // int | The byte offset this chunk starts at.
$recording_id = 56; // int | The numeric id from a recording index row.
$upload_length = 'upload_length_example'; // string | The total size of the artifact, or `*` while it is unknown.
$upload_complete = 'upload_complete_example'; // string | Set to `1` when this chunk ends the upload.
$upload_probe = 'upload_probe_example'; // string | Set to `1` to ask for the current offset without staging any bytes. The generated clients use this instead of an empty body.
$body = '/path/to/file.txt'; // \SplFileObject

try {
    $result = $apiInstance->appendRecordingArtifact($ext, $upload_offset, $recording_id, $upload_length, $upload_complete, $upload_probe, $body);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling RecordingsApi->appendRecordingArtifact: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **ext** | **string**| The artifact extension, allow-listed exactly like the raw-body POST shape. | |
| **upload_offset** | **int**| The byte offset this chunk starts at. | |
| **recording_id** | **int**| The numeric id from a recording index row. | |
| **upload_length** | **string**| The total size of the artifact, or &#x60;*&#x60; while it is unknown. | [optional] |
| **upload_complete** | **string**| Set to &#x60;1&#x60; when this chunk ends the upload. | [optional] |
| **upload_probe** | **string**| Set to &#x60;1&#x60; to ask for the current offset without staging any bytes. The generated clients use this instead of an empty body. | [optional] |
| **body** | **\SplFileObject****\SplFileObject**|  | [optional] |

### Return type

[**\OpenAPI\Client\Model\SessionRecording**](../Model/SessionRecording.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/octet-stream`
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `createRecordingArtifactTicket()`

```php
createRecordingArtifactTicket($recording_id): \OpenAPI\Client\Model\PlaybackTicket
```

Mint a short-lived playback URL for a recording

Owner-only. Returns a signed, time-limited `url` that can be fetched **without** an `Authorization` header, which is what makes playback work from `<video src>`. The ticket is bound to the recording, the owner and its purpose, so an artifact ticket cannot be replayed against the thumbnail route. Minting is stateless - nothing is written - and the link stops working when `expires_at` passes. `400` when playback tickets are disabled on the deployment (`RECORDING_URL_SECRET`/`API_KEY` unset); `404` when the recording has no stored artifact.

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


$apiInstance = new OpenAPI\Client\Api\RecordingsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$recording_id = 56; // int | The numeric id from a recording index row.

try {
    $result = $apiInstance->createRecordingArtifactTicket($recording_id);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling RecordingsApi->createRecordingArtifactTicket: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **recording_id** | **int**| The numeric id from a recording index row. | |

### Return type

[**\OpenAPI\Client\Model\PlaybackTicket**](../Model/PlaybackTicket.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `createRecordingThumbnailTicket()`

```php
createRecordingThumbnailTicket($recording_id): \OpenAPI\Client\Model\PlaybackTicket
```

Mint a short-lived preview URL for a recording

Same rules as the artifact ticket, but the URL addresses the thumbnail route and the ticket's purpose is checked on redemption.

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


$apiInstance = new OpenAPI\Client\Api\RecordingsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$recording_id = 56; // int | The numeric id from a recording index row.

try {
    $result = $apiInstance->createRecordingThumbnailTicket($recording_id);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling RecordingsApi->createRecordingThumbnailTicket: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **recording_id** | **int**| The numeric id from a recording index row. | |

### Return type

[**\OpenAPI\Client\Model\PlaybackTicket**](../Model/PlaybackTicket.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `createSessionRecording()`

```php
createSessionRecording($session_recording_input): \OpenAPI\Client\Model\SessionRecording
```

Index a new recording

Called by the capturing side when a session starts (or when a finished capture is handed over), so the owner sees the recording while it is still being written and the row can carry the outcome afterwards. `status` is one of `recording`, `completed` or `failed`; `started_at` defaults to now. The media is uploaded separately to `.../artifact`.  The row belongs to the *caller*: `device_id` is recorded as a label and is not checked for ownership, so a producer can index a capture for a machine that is not bound to it. Only the caller can then read or delete the row.

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


$apiInstance = new OpenAPI\Client\Api\RecordingsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$session_recording_input = new \OpenAPI\Client\Model\SessionRecordingInput(); // \OpenAPI\Client\Model\SessionRecordingInput

try {
    $result = $apiInstance->createSessionRecording($session_recording_input);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling RecordingsApi->createSessionRecording: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **session_recording_input** | [**\OpenAPI\Client\Model\SessionRecordingInput**](../Model/SessionRecordingInput.md)|  | |

### Return type

[**\OpenAPI\Client\Model\SessionRecording**](../Model/SessionRecording.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `deleteSessionRecording()`

```php
deleteSessionRecording($recording_id)
```

Delete a recording and its stored media

Removes the index row and the media files that belong to it, so the recording cannot be played back afterwards. There is no \"already gone\" success: a second delete is a `404`. The deletion is published on the owner's realtime stream and written to the audit trail as `recording.delete` with the actor's observed address.

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


$apiInstance = new OpenAPI\Client\Api\RecordingsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$recording_id = 56; // int | The numeric id from a recording index row.

try {
    $apiInstance->deleteSessionRecording($recording_id);
} catch (Exception $e) {
    echo 'Exception when calling RecordingsApi->deleteSessionRecording: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **recording_id** | **int**| The numeric id from a recording index row. | |

### Return type

void (empty response body)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `downloadRecordingArtifact()`

```php
downloadRecordingArtifact($recording_id): \SplFileObject
```

Download a recording's media bytes

Owner-only, `Authorization` (Bearer) required. The response is the stored file itself: the `Content-Type` follows the stored file's extension (`video/mp4` for `.mp4`, `application/octet-stream` otherwise) and `Content-Disposition` is `inline`. A browser cannot attach an `Authorization` header to `<video src>`, which is why the ticket routes below exist; this route is for API clients.

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


$apiInstance = new OpenAPI\Client\Api\RecordingsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$recording_id = 56; // int | The numeric id from a recording index row.

try {
    $result = $apiInstance->downloadRecordingArtifact($recording_id);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling RecordingsApi->downloadRecordingArtifact: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **recording_id** | **int**| The numeric id from a recording index row. | |

### Return type

**\SplFileObject**

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `video/mp4`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `downloadRecordingArtifactByTicket()`

```php
downloadRecordingArtifactByTicket($t, $recording_id): \SplFileObject
```

Fetch recording media with a playback ticket

The URL returned by `POST /v1/me/recordings/{recording_id}/artifact-ticket`. Deliberately registered **outside** the API-key and user-auth groups, because a browser `<video>` element can send neither header - the signed `t` query parameter is the authorization, and the ticket is bound to owner and purpose. `401` for a missing, malformed or expired ticket; `404` when the ticket is for the wrong route or the recording is gone.

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


$apiInstance = new OpenAPI\Client\Api\RecordingsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$t = 't_example'; // string | The signed ticket from the artifact-ticket endpoint.
$recording_id = 56; // int | The numeric id from a recording index row.

try {
    $result = $apiInstance->downloadRecordingArtifactByTicket($t, $recording_id);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling RecordingsApi->downloadRecordingArtifactByTicket: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **t** | **string**| The signed ticket from the artifact-ticket endpoint. | |
| **recording_id** | **int**| The numeric id from a recording index row. | |

### Return type

**\SplFileObject**

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `video/mp4`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `downloadRecordingThumbnail()`

```php
downloadRecordingThumbnail($recording_id): \SplFileObject
```

Download a recording's preview image

Owner-only. Served with `Cache-Control: private, max-age=300` because a preview is user-scoped and can change under the same URL.

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


$apiInstance = new OpenAPI\Client\Api\RecordingsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$recording_id = 56; // int | The numeric id from a recording index row.

try {
    $result = $apiInstance->downloadRecordingThumbnail($recording_id);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling RecordingsApi->downloadRecordingThumbnail: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **recording_id** | **int**| The numeric id from a recording index row. | |

### Return type

**\SplFileObject**

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `image/jpeg`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `downloadRecordingThumbnailByTicket()`

```php
downloadRecordingThumbnailByTicket($t, $recording_id): \SplFileObject
```

Fetch a recording preview with a playback ticket

The URL returned by `POST /v1/me/recordings/{recording_id}/thumbnail-ticket`. Same ticket rules as the artifact route; an artifact ticket is rejected here (`404`) because the purpose is part of the signature.

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


$apiInstance = new OpenAPI\Client\Api\RecordingsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$t = 't_example'; // string | The signed ticket from the thumbnail-ticket endpoint.
$recording_id = 56; // int | The numeric id from a recording index row.

try {
    $result = $apiInstance->downloadRecordingThumbnailByTicket($t, $recording_id);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling RecordingsApi->downloadRecordingThumbnailByTicket: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **t** | **string**| The signed ticket from the thumbnail-ticket endpoint. | |
| **recording_id** | **int**| The numeric id from a recording index row. | |

### Return type

**\SplFileObject**

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `image/jpeg`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `getSessionRecording()`

```php
getSessionRecording($recording_id): \OpenAPI\Client\Model\SessionRecording
```

One recording the caller owns

The same row the list returns. Another user's id is a `404` rather than a `403`, so the route cannot be used to discover which recording ids exist.

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


$apiInstance = new OpenAPI\Client\Api\RecordingsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$recording_id = 56; // int | The numeric id from a recording index row.

try {
    $result = $apiInstance->getSessionRecording($recording_id);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling RecordingsApi->getSessionRecording: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **recording_id** | **int**| The numeric id from a recording index row. | |

### Return type

[**\OpenAPI\Client\Model\SessionRecording**](../Model/SessionRecording.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `listSessionRecordings()`

```php
listSessionRecordings($cursor, $limit): \OpenAPI\Client\Model\SessionRecordingList
```

The caller's session-recording index

Cursor-paginated, newest first. `next_cursor` is opaque: pass it back as `?cursor=` to fetch the next page and stop when it is absent. The envelope here has no `total` because this endpoint does not count the whole set on every page, so an integrator must page until `next_cursor` is empty rather than trusting a count. This is the metadata index only; the media bytes live behind the per-recording `.../artifact` routes documented in `docs/user-api-docs.md`.

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


$apiInstance = new OpenAPI\Client\Api\RecordingsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$cursor = 'cursor_example'; // string
$limit = 56; // int

try {
    $result = $apiInstance->listSessionRecordings($cursor, $limit);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling RecordingsApi->listSessionRecordings: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **cursor** | **string**|  | [optional] |
| **limit** | **int**|  | [optional] |

### Return type

[**\OpenAPI\Client\Model\SessionRecordingList**](../Model/SessionRecordingList.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `updateSessionRecording()`

```php
updateSessionRecording($recording_id, $session_recording_patch): \OpenAPI\Client\Model\SessionRecording
```

Update a recording's lifecycle metadata

Partial update: only the fields present in the body are written, so a producer can patch `status` when a capture starts and then `size_bytes`/`ended_at` when the upload finishes. `status` accepts the same three values as create; anything else is a `400` and writes nothing.

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


$apiInstance = new OpenAPI\Client\Api\RecordingsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$recording_id = 56; // int | The numeric id from a recording index row.
$session_recording_patch = new \OpenAPI\Client\Model\SessionRecordingPatch(); // \OpenAPI\Client\Model\SessionRecordingPatch

try {
    $result = $apiInstance->updateSessionRecording($recording_id, $session_recording_patch);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling RecordingsApi->updateSessionRecording: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **recording_id** | **int**| The numeric id from a recording index row. | |
| **session_recording_patch** | [**\OpenAPI\Client\Model\SessionRecordingPatch**](../Model/SessionRecordingPatch.md)|  | |

### Return type

[**\OpenAPI\Client\Model\SessionRecording**](../Model/SessionRecording.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `uploadRecordingArtifact()`

```php
uploadRecordingArtifact($recording_id, $file, $ext): \OpenAPI\Client\Model\SessionRecording
```

Upload a recording's media bytes

Two request shapes, because the producers differ: `multipart/form-data` with a `file` part (the browser and Qt clients) or the raw body with `?ext=`. The extension is allow-listed (`.mp4`, `.m4v`, `.mkv`, `.webm`, `.h264`, `.grf`), the bytes are written to a temporary file and renamed into place so a dropped upload can never leave a half-written file under the real name, and the row is then updated with `artifact_path`, `artifact_url`, `format` and `size_bytes` in one step.  An empty body is a `400`; a body over the configured limit (`RECORDING_MAX_BYTES`, exposed as `store.max_bytes`) is a `400` as well, never a truncated file. Uploading again replaces the stored artifact.

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


$apiInstance = new OpenAPI\Client\Api\RecordingsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$recording_id = 56; // int | The numeric id from a recording index row.
$file = '/path/to/file.txt'; // \SplFileObject
$ext = 'ext_example'; // string | Required for the raw-body shape; ignored for multipart.

try {
    $result = $apiInstance->uploadRecordingArtifact($recording_id, $file, $ext);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling RecordingsApi->uploadRecordingArtifact: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **recording_id** | **int**| The numeric id from a recording index row. | |
| **file** | **\SplFileObject****\SplFileObject**|  | |
| **ext** | **string**| Required for the raw-body shape; ignored for multipart. | [optional] |

### Return type

[**\OpenAPI\Client\Model\SessionRecording**](../Model/SessionRecording.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `multipart/form-data`, `application/octet-stream`
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)

## `uploadRecordingThumbnail()`

```php
uploadRecordingThumbnail($recording_id, $file, $ext): \OpenAPI\Client\Model\SessionRecording
```

Upload a recording's preview image

The same two shapes as the artifact upload (`multipart/form-data` with a `file` part, or the raw body with `?ext=`), but the extension is limited to `.jpg`, `.jpeg` and `.png` and the size to 2 MiB - a preview is a small still, and an oversized one means the wrong file was sent.  The bytes are sniffed before they are stored rather than trusted from the name or the declared content type: the index renders this image inline, so a file that only claims to be a JPEG is a `400`.

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


$apiInstance = new OpenAPI\Client\Api\RecordingsApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$recording_id = 56; // int | The numeric id from a recording index row.
$file = '/path/to/file.txt'; // \SplFileObject
$ext = 'ext_example'; // string | Required for the raw-body shape; ignored for multipart.

try {
    $result = $apiInstance->uploadRecordingThumbnail($recording_id, $file, $ext);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling RecordingsApi->uploadRecordingThumbnail: ', $e->getMessage(), PHP_EOL;
}
```

### Parameters

| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **recording_id** | **int**| The numeric id from a recording index row. | |
| **file** | **\SplFileObject****\SplFileObject**|  | |
| **ext** | **string**| Required for the raw-body shape; ignored for multipart. | [optional] |

### Return type

[**\OpenAPI\Client\Model\SessionRecording**](../Model/SessionRecording.md)

### Authorization

[ApiKeyAuth](../../README.md#ApiKeyAuth), [BearerAuth](../../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `multipart/form-data`, `application/octet-stream`
- **Accept**: `application/json`, `application/problem+json`

[[Back to top]](#) [[Back to API list]](../../README.md#endpoints)
[[Back to Model list]](../../README.md#models)
[[Back to README]](../../README.md)
