# QuickDesk.Signaling.Client.Model.NotificationChannel

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **long** |  | 
**UserId** | **long** |  | 
**Name** | **string** |  | 
**Url** | **string** |  | 
**Events** | **string** | A JSON-encoded array of subscribed event types (a string, not an array, because that is how the column is stored). Decode it before matching against &#x60;supported_events&#x60;.  | 
**Enabled** | **bool** |  | 
**LastTriggered** | **DateTime?** |  | [optional] 
**LastStatus** | **int** | HTTP status of the most recent delivery attempt; 0 means none yet. | 
**CreatedAt** | **DateTime** |  | [optional] 
**UpdatedAt** | **DateTime** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

