# # RemoteSessionBatchRevocation

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**device_id** | **string** |  |
**revoked** | **int** | How many sessions were closed and confirmed. |
**failed** | **int** | How many were live but could not be closed. |
**truncated** | **bool** | True when more live sessions existed than one call handles (the cap is 200). |
**sessions** | [**\OpenAPI\Client\Model\RemoteSessionBatchRevocationSessionsInner[]**](RemoteSessionBatchRevocationSessionsInner.md) | One entry per session the call attempted, newest first. |

[[Back to Model list]](../../README.md#models) [[Back to API list]](../../README.md#endpoints) [[Back to README]](../../README.md)
