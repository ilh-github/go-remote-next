# QuickDesk.Signaling.Client.Model.RemoteSessionBatchRevocation
The result of clearing every live control session off one device. Unlike the single-session response this is a *report*: a batch can half succeed, so the counts and the per-session entries are the answer. `truncated` is true when the device had more live controllers than one call handles, so a caller that sees it knows the machine may not be empty yet. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DeviceId** | **string** |  | 
**Revoked** | **int** | How many sessions were closed and confirmed. | 
**Failed** | **int** | How many were live but could not be closed. | 
**Truncated** | **bool** | True when more live sessions existed than one call handles (the cap is 200). | 
**Sessions** | [**List&lt;RemoteSessionBatchRevocationSessionsInner&gt;**](RemoteSessionBatchRevocationSessionsInner.md) | One entry per session the call attempted, newest first. | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

