
# RemoteSessionBatchRevocation

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **deviceId** | **kotlin.String** |  |  |
| **revoked** | **kotlin.Int** | How many sessions were closed and confirmed. |  |
| **failed** | **kotlin.Int** | How many were live but could not be closed. |  |
| **truncated** | **kotlin.Boolean** | True when more live sessions existed than one call handles (the cap is 200). |  |
| **sessions** | [**kotlin.collections.List&lt;RemoteSessionBatchRevocationSessionsInner&gt;**](RemoteSessionBatchRevocationSessionsInner.md) | One entry per session the call attempted, newest first. |  |



