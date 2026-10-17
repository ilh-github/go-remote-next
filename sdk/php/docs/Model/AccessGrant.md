# # AccessGrant

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **int** |  |
**device_id** | **string** |  |
**code** | **string** | Returned only in the creation response; treat it as a secret. |
**label** | **string** |  | [optional]
**expires_at** | **\DateTime** |  |
**max_uses** | **int** |  |
**uses** | **int** |  |
**active** | **bool** |  |
**revoked_at** | **\DateTime** |  | [optional]
**last_used_at** | **\DateTime** |  | [optional]
**bound_client_id** | **string** |  |
**bound_ips** | **string[]** |  |
**last_used_client_id** | **string** |  | [optional]
**last_used_ip** | **string** |  | [optional]
**created_at** | **\DateTime** |  |

[[Back to Model list]](../../README.md#models) [[Back to API list]](../../README.md#endpoints) [[Back to README]](../../README.md)
