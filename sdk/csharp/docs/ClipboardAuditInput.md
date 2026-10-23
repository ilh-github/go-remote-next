# QuickDesk.Signaling.Client.Model.ClipboardAuditInput
A controller report about one clipboard sync that happened outside the signaling data path. The server stores the action as `clipboard.sync.upload` or `clipboard.sync.download`, the resource as `clipboard_sync/{transfer_id}`, and the fields below as JSON details.  There is intentionally no field for the clipboard text. See the endpoint description for why that is a property of the contract rather than an omission from it. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DeviceId** | **string** | A device owned by the authenticated caller. | 
**TransferId** | **string** | Stable id used to make retries idempotent. | 
**Direction** | **string** | Which way the clipboard went: upload is controller to host, download is host to controller. | 
**Status** | **string** |  | 
**MimeType** | **string** | The clipboard format, e.g. &#x60;text/plain&#x60;. Metadata only. | 
**Bytes** | **long** | Size of the payload that crossed the data channel. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

