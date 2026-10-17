
# AccessApproval

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **id** | **kotlin.Long** |  |  |
| **deviceId** | **kotlin.String** |  |  |
| **clientId** | **kotlin.String** | Self-reported by the requesting controller. |  |
| **clientIp** | **kotlin.String** | The address the server observed (trusted-proxy aware). |  |
| **source** | [**inline**](#Source) |  |  |
| **status** | [**inline**](#Status) |  |  |
| **expiresAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  |
| **createdAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  |
| **grantId** | **kotlin.Long** |  |  [optional] |
| **decidedAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  [optional] |
| **reason** | **kotlin.String** |  |  [optional] |


<a id="Source"></a>
## Enum: source
| Name | Value |
| ---- | ----- |
| source | device_code, grant |


<a id="Status"></a>
## Enum: status
| Name | Value |
| ---- | ----- |
| status | pending, approved, denied, expired, consumed |



