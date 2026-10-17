# QuickDesk.Signaling.Client.Model.AccessCodeVerifyResponse
The answer to `access-code:verify` and to the approval claim endpoint. `status` says which shape it is: `granted` (HTTP 200) carries `signal_token`, `pending` (HTTP 202) carries the approval the controller polls. The approval fields live on this schema rather than only on the 202 schema because a generated client declares one success type per operation - a Go caller receives the 202 body as this type, and without these properties it could not read the approval it is supposed to poll. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Status** | **string** | Which shape this response is. | 
**SignalToken** | **string** | Present when &#x60;status&#x60; is &#x60;granted&#x60;. | [optional] 
**ExpiresAt** | **DateTime** | When the signal token (granted) or the approval (pending) expires. | [optional] 
**ApprovalId** | **long** | Present when &#x60;status&#x60; is &#x60;pending&#x60;. | [optional] 
**ApprovalSecret** | **string** | Present when &#x60;status&#x60; is &#x60;pending&#x60;. Shown once; exchanged for a signal token at the claim endpoint.  | [optional] 
**PollAfterMs** | **int** | Present when &#x60;status&#x60; is &#x60;pending&#x60;. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

