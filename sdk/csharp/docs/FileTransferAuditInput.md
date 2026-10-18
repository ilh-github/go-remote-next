# QuickDesk.Signaling.Client.Model.FileTransferAuditInput
A controller report about one file transferred outside the signaling data path. The server stores the action as `file.transfer.upload` or `file.transfer.download`, the resource as `file_transfer/{transfer_id}`, and the remaining fields as JSON details. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DeviceId** | **string** | A device owned by the authenticated caller. | 
**TransferId** | **string** | Stable id used to make retries idempotent. | 
**Direction** | **string** | Upload is controller to host; download is host to controller. | 
**Status** | **string** |  | 
**Filename** | **string** |  | [optional] 
**BytesTransferred** | **long** |  | [optional] 
**TotalBytes** | **long** | Zero means the controller did not know the total size. | [optional] 
**Error** | **string** | Human-readable failure detail when status is failed. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

