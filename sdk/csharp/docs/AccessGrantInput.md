# QuickDesk.Signaling.Client.Model.AccessGrantInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Label** | **string** |  | [optional] 
**ExpiresInMinutes** | **int** |  | 
**MaxUses** | **int** | 0 means \&quot;no use limit\&quot;; the grant still expires. | [optional] 
**Code** | **string** | Optional caller-chosen code; omitted means the server generates one. | [optional] 
**BoundClientId** | **string** | Self-reported controller identity the code is bound to. | [optional] 
**BoundIps** | **List&lt;string&gt;** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

