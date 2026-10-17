
# AccessGrant

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **id** | **kotlin.Long** |  |  |
| **deviceId** | **kotlin.String** |  |  |
| **code** | **kotlin.String** | Returned only in the creation response; treat it as a secret. |  |
| **expiresAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  |
| **maxUses** | **kotlin.Int** |  |  |
| **uses** | **kotlin.Int** |  |  |
| **active** | **kotlin.Boolean** |  |  |
| **boundClientId** | **kotlin.String** |  |  |
| **boundIps** | **kotlin.collections.List&lt;kotlin.String&gt;** |  |  |
| **createdAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  |
| **label** | **kotlin.String** |  |  [optional] |
| **revokedAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  [optional] |
| **lastUsedAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  [optional] |
| **lastUsedClientId** | **kotlin.String** |  |  [optional] |
| **lastUsedIp** | **kotlin.String** |  |  [optional] |



