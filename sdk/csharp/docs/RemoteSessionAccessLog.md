# QuickDesk.Signaling.Client.Model.RemoteSessionAccessLog
One device access-log row that carries a session id - in practice the `session_started` entry the server wrote when the controller authenticated, which is what pairs \"an attempt\" with \"the session it became\". The full timeline is documented separately in `docs/user-api-docs.md` (§5.3.1); this is the part a session owns. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **long** |  | 
**DeviceId** | **string** |  | 
**OwnerId** | **long** | The device&#39;s owner at the time of the attempt; 0 when unbound. | 
**ClientId** | **string** |  | 
**SessionId** | **string** |  | 
**Ip** | **string** |  | 
**Result** | **string** | The canonical access result, e.g. &#x60;session_started&#x60;. | 
**Reason** | **string** |  | 
**CreatedAt** | **DateTime** |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

