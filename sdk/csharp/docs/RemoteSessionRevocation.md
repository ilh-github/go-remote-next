# QuickDesk.Signaling.Client.Model.RemoteSessionRevocation
The result of ending one live control session. `revoked` is always `true` - the endpoint answers `409` rather than returning a false here, so a caller never has to interpret a success response as \"maybe\". 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**SessionId** | **string** |  | 
**DeviceId** | **string** |  | 
**Revoked** | **bool** |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

