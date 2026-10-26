# quickdesk_signaling.FileTransfersApi

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**create_clipboard_audit**](FileTransfersApi.md#create_clipboard_audit) | **POST** /v1/me/clipboard-audits | Report a controller-observed clipboard sync for auditing
[**create_file_transfer_audit**](FileTransfersApi.md#create_file_transfer_audit) | **POST** /v1/me/file-transfer-audits | Report a controller-observed file transfer for auditing
[**list_my_audit_logs**](FileTransfersApi.md#list_my_audit_logs) | **GET** /v1/me/audit-logs | Read back the data-channel audit trail that belongs to the caller


# **create_clipboard_audit**
> create_clipboard_audit(clipboard_audit_input)

Report a controller-observed clipboard sync for auditing

Clipboard bytes travel directly over the Remoting data channel and never
pass through the signaling server. This endpoint lets an authenticated
device owner report a clipboard sync after the fact so it appears in the
audit trail.

The report carries **metadata only**: direction, MIME type and byte
count. There is deliberately no field for the clipboard text. A clipboard
is by definition whatever the user last copied - routinely a password, a
recovery code or a private message - so a row that stored the text would
turn the audit trail into a credential store every later reader of the
database inherits. "The operator copied 1.2 KB of text at 14:03" answers
the investigative question; the text answers nothing an investigator
needs and costs everything if the database leaks.

Reports are idempotent by `transfer_id`, and the device must belong to
the caller. This is an attributable controller report, not cryptographic
proof from the host that a clipboard sync happened.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.clipboard_audit_input import ClipboardAuditInput
from quickdesk_signaling.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://your-signaling-host
# See configuration.py for a list of all supported configuration parameters.
configuration = quickdesk_signaling.Configuration(
    host = "https://your-signaling-host"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Configure Bearer authorization (opaque): BearerAuth
configuration = quickdesk_signaling.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with quickdesk_signaling.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = quickdesk_signaling.FileTransfersApi(api_client)
    clipboard_audit_input = quickdesk_signaling.ClipboardAuditInput() # ClipboardAuditInput | 

    try:
        # Report a controller-observed clipboard sync for auditing
        api_instance.create_clipboard_audit(clipboard_audit_input)
    except Exception as e:
        print("Exception when calling FileTransfersApi->create_clipboard_audit: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **clipboard_audit_input** | [**ClipboardAuditInput**](ClipboardAuditInput.md)|  | 

### Return type

void (empty response body)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**204** | The clipboard report is in the audit trail (or was already recorded). |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |
**403** | Authenticated but not allowed (includes a rejected access code). |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **create_file_transfer_audit**
> create_file_transfer_audit(file_transfer_audit_input)

Report a controller-observed file transfer for auditing

File bytes travel directly over the Remoting data channel and never
pass through the signaling server. This endpoint lets an authenticated
device owner report a completed or failed transfer after the fact so it
appears in the audit trail.

Reports are idempotent by `(actor, direction, transfer_id)`: retrying
the same event does not create another row. The device must belong to
the caller. This is an attributable controller report, not
cryptographic proof from the host that a file operation happened.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.file_transfer_audit_input import FileTransferAuditInput
from quickdesk_signaling.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://your-signaling-host
# See configuration.py for a list of all supported configuration parameters.
configuration = quickdesk_signaling.Configuration(
    host = "https://your-signaling-host"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Configure Bearer authorization (opaque): BearerAuth
configuration = quickdesk_signaling.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with quickdesk_signaling.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = quickdesk_signaling.FileTransfersApi(api_client)
    file_transfer_audit_input = quickdesk_signaling.FileTransferAuditInput() # FileTransferAuditInput | 

    try:
        # Report a controller-observed file transfer for auditing
        api_instance.create_file_transfer_audit(file_transfer_audit_input)
    except Exception as e:
        print("Exception when calling FileTransfersApi->create_file_transfer_audit: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **file_transfer_audit_input** | [**FileTransferAuditInput**](FileTransferAuditInput.md)|  | 

### Return type

void (empty response body)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**204** | The transfer report is in the audit trail (or was already recorded). |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |
**403** | Authenticated but not allowed (includes a rejected access code). |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_my_audit_logs**
> OwnerAuditLogPage list_my_audit_logs(action=action, cursor=cursor, limit=limit)

Read back the data-channel audit trail that belongs to the caller

The read side of the two reporting endpoints above. File and clipboard
transfers travel peer to peer over the Remoting data channel, so the
control plane can only record what a controller tells it afterwards;
this is how the device owner reads those records back without going
through the administrator console.

Rows are scoped to the caller's own actor id. Both reporting endpoints
refuse a device the caller does not own, so the actor on a row is that
device's owner and this scope cannot expose anybody else's transfers.

`action` is an allow-list rather than a free-form filter: an action this
trail does not hold is refused with 400 rather than answered with an
empty page, because "you did nothing" must not be a conclusion a typo
can produce. Supported values are `file.transfer.upload`,
`file.transfer.download`, `clipboard.sync.upload` and
`clipboard.sync.download`.

Cursor-paginated, newest first. `next_cursor` is opaque: pass it back as
`?cursor=` to fetch the next page and stop when it is absent.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.owner_audit_log_page import OwnerAuditLogPage
from quickdesk_signaling.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://your-signaling-host
# See configuration.py for a list of all supported configuration parameters.
configuration = quickdesk_signaling.Configuration(
    host = "https://your-signaling-host"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Configure Bearer authorization (opaque): BearerAuth
configuration = quickdesk_signaling.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with quickdesk_signaling.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = quickdesk_signaling.FileTransfersApi(api_client)
    action = 'action_example' # str | Narrow to one audit action. An unsupported value is a 400. (optional)
    cursor = 'cursor_example' # str |  (optional)
    limit = 56 # int |  (optional)

    try:
        # Read back the data-channel audit trail that belongs to the caller
        api_response = api_instance.list_my_audit_logs(action=action, cursor=cursor, limit=limit)
        print("The response of FileTransfersApi->list_my_audit_logs:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling FileTransfersApi->list_my_audit_logs: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **action** | **str**| Narrow to one audit action. An unsupported value is a 400. | [optional] 
 **cursor** | **str**|  | [optional] 
 **limit** | **int**|  | [optional] 

### Return type

[**OwnerAuditLogPage**](OwnerAuditLogPage.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | One page of the caller&#39;s audit rows, newest first. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

