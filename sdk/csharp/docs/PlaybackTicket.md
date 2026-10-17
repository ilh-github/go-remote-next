# QuickDesk.Signaling.Client.Model.PlaybackTicket
A signed, time-limited URL for one recording's media or preview. It carries no credentials, so it must be treated as a bearer token: anyone who holds the link can fetch the file until `expires_at`. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Url** | **string** | Absolute path (with the signed &#x60;t&#x60; parameter) to fetch. | 
**ExpiresAt** | **DateTime** |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

