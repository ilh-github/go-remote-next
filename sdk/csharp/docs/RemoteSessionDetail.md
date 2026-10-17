# QuickDesk.Signaling.Client.Model.RemoteSessionDetail
One control session with the records that share its `session_id`. Both lists are capped at 100 entries and present-but-empty when there is nothing to show, so a client never has to distinguish \"no recordings\" from \"field missing\". 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Session** | [**RemoteSession**](RemoteSession.md) |  | 
**Recordings** | [**List&lt;SessionRecording&gt;**](SessionRecording.md) |  | 
**AccessLog** | [**List&lt;RemoteSessionAccessLog&gt;**](RemoteSessionAccessLog.md) |  | 
**Counts** | [**RemoteSessionDetailCounts**](RemoteSessionDetailCounts.md) |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

