# FileTransferAuditInput

A controller report about one file transferred outside the signaling data path. The server stores the action as `file.transfer.upload` or `file.transfer.download`, the resource as `file_transfer/{transfer_id}`, and the remaining fields as JSON details. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**device_id** | **str** | A device owned by the authenticated caller. | 
**transfer_id** | **str** | Stable id used to make retries idempotent. | 
**direction** | **str** | Upload is controller to host; download is host to controller. | 
**status** | **str** |  | 
**filename** | **str** |  | [optional] 
**bytes_transferred** | **int** |  | [optional] 
**total_bytes** | **int** | Zero means the controller did not know the total size. | [optional] 
**error** | **str** | Human-readable failure detail when status is failed. | [optional] 

## Example

```python
from quickdesk_signaling.models.file_transfer_audit_input import FileTransferAuditInput

# TODO update the JSON string below
json = "{}"
# create an instance of FileTransferAuditInput from a JSON string
file_transfer_audit_input_instance = FileTransferAuditInput.from_json(json)
# print the JSON string representation of the object
print(FileTransferAuditInput.to_json())

# convert the object into a dict
file_transfer_audit_input_dict = file_transfer_audit_input_instance.to_dict()
# create an instance of FileTransferAuditInput from a dict
file_transfer_audit_input_from_dict = FileTransferAuditInput.from_dict(file_transfer_audit_input_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


