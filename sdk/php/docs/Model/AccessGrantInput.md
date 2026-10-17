# # AccessGrantInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**label** | **string** |  | [optional]
**expires_in_minutes** | **int** |  |
**max_uses** | **int** | 0 means \&quot;no use limit\&quot;; the grant still expires. | [optional]
**code** | **string** | Optional caller-chosen code; omitted means the server generates one. | [optional]
**bound_client_id** | **string** | Self-reported controller identity the code is bound to. | [optional]
**bound_ips** | **string[]** |  | [optional]

[[Back to Model list]](../../README.md#models) [[Back to API list]](../../README.md#endpoints) [[Back to README]](../../README.md)
