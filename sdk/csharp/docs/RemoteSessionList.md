# QuickDesk.Signaling.Client.Model.RemoteSessionList
Cursor page of control sessions. `next_cursor` is absent on the last page and `total` is deliberately absent, the same as every other paginated list here: the endpoint does not count the whole set per page. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Items** | [**List&lt;RemoteSession&gt;**](RemoteSession.md) |  | 
**NextCursor** | **string** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

