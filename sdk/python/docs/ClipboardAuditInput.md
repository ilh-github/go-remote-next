# ClipboardAuditInput

A controller report about one clipboard sync that happened outside the signaling data path. The server stores the action as `clipboard.sync.upload` or `clipboard.sync.download`, the resource as `clipboard_sync/{transfer_id}`, and the fields below as JSON details.  There is intentionally no field for the clipboard text. See the endpoint description for why that is a property of the contract rather than an omission from it. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**device_id** | **str** | A device owned by the authenticated caller. | 
**transfer_id** | **str** | Stable id used to make retries idempotent. | 
**direction** | **str** | Which way the clipboard went: upload is controller to host, download is host to controller. | 
**status** | **str** |  | 
**mime_type** | **str** | The clipboard format, e.g. &#x60;text/plain&#x60;. Metadata only. | 
**bytes** | **int** | Size of the payload that crossed the data channel. | [optional] 

## Example

```python
from quickdesk_signaling.models.clipboard_audit_input import ClipboardAuditInput

# TODO update the JSON string below
json = "{}"
# create an instance of ClipboardAuditInput from a JSON string
clipboard_audit_input_instance = ClipboardAuditInput.from_json(json)
# print the JSON string representation of the object
print(ClipboardAuditInput.to_json())

# convert the object into a dict
clipboard_audit_input_dict = clipboard_audit_input_instance.to_dict()
# create an instance of ClipboardAuditInput from a dict
clipboard_audit_input_from_dict = ClipboardAuditInput.from_dict(clipboard_audit_input_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


