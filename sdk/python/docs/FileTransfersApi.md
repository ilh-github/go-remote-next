# quickdesk_signaling.FileTransfersApi

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**create_file_transfer_audit**](FileTransfersApi.md#create_file_transfer_audit) | **POST** /v1/me/file-transfer-audits | Report a controller-observed file transfer for auditing


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

