
# AccessGrantInput

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **expiresInMinutes** | **kotlin.Int** |  |  |
| **label** | **kotlin.String** |  |  [optional] |
| **maxUses** | **kotlin.Int** | 0 means \&quot;no use limit\&quot;; the grant still expires. |  [optional] |
| **code** | **kotlin.String** | Optional caller-chosen code; omitted means the server generates one. |  [optional] |
| **boundClientId** | **kotlin.String** | Self-reported controller identity the code is bound to. |  [optional] |
| **boundIps** | **kotlin.collections.List&lt;kotlin.String&gt;** |  |  [optional] |



