# QuickDesk.Signaling.Client.Model.AccessCodeApprovalPending
The HTTP 202 body of `access-code:verify` - the same fields as `AccessCodeVerifyResponse` with `status: pending`. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ApprovalId** | **long** |  | 
**ApprovalSecret** | **string** | Shown once; exchanged for a signal token at the claim endpoint. | 
**Status** | **string** |  | 
**ExpiresAt** | **DateTime** |  | 
**PollAfterMs** | **int** |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

