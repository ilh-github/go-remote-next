
# AccessCodeApprovalPending

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **approvalId** | **kotlin.Long** |  |  |
| **approvalSecret** | **kotlin.String** | Shown once; exchanged for a signal token at the claim endpoint. |  |
| **status** | [**inline**](#Status) |  |  |
| **expiresAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  |
| **pollAfterMs** | **kotlin.Int** |  |  |


<a id="Status"></a>
## Enum: status
| Name | Value |
| ---- | ----- |
| status | pending |



