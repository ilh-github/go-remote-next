
# AccessCodeVerifyResponse

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **status** | [**inline**](#Status) | Which shape this response is. |  |
| **signalToken** | **kotlin.String** | Present when &#x60;status&#x60; is &#x60;granted&#x60;. |  [optional] |
| **expiresAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) | When the signal token (granted) or the approval (pending) expires. |  [optional] |
| **approvalId** | **kotlin.Long** | Present when &#x60;status&#x60; is &#x60;pending&#x60;. |  [optional] |
| **approvalSecret** | **kotlin.String** | Present when &#x60;status&#x60; is &#x60;pending&#x60;. Shown once; exchanged for a signal token at the claim endpoint.  |  [optional] |
| **pollAfterMs** | **kotlin.Int** | Present when &#x60;status&#x60; is &#x60;pending&#x60;. |  [optional] |


<a id="Status"></a>
## Enum: status
| Name | Value |
| ---- | ----- |
| status | granted, pending |



