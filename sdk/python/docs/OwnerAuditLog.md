# OwnerAuditLog

One audit row. `admin_id` and `admin_username` are always empty here: this trail is written by end-user actors, and an administrator acting on the same resource lands in the admin-side trail instead. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **int** |  | 
**actor_user_id** | **int** | The account that reported the transfer. Always the caller. | [optional] 
**action** | **str** | One of &#x60;file.transfer.upload&#x60;, &#x60;file.transfer.download&#x60;, &#x60;clipboard.sync.upload&#x60;, &#x60;clipboard.sync.download&#x60;.  | 
**resource_type** | **str** | &#x60;file_transfer&#x60; or &#x60;clipboard_sync&#x60;.  | 
**resource_id** | **str** | The transfer id the reporter supplied, which is the idempotency key. | 
**details** | **str** | JSON-encoded metadata: byte counts, direction, status and, for file transfers, the filename. Never the transferred content itself.  | [optional] 
**ip** | **str** | The source address the report arrived from. | [optional] 
**created_at** | **datetime** |  | 

## Example

```python
from quickdesk_signaling.models.owner_audit_log import OwnerAuditLog

# TODO update the JSON string below
json = "{}"
# create an instance of OwnerAuditLog from a JSON string
owner_audit_log_instance = OwnerAuditLog.from_json(json)
# print the JSON string representation of the object
print(OwnerAuditLog.to_json())

# convert the object into a dict
owner_audit_log_dict = owner_audit_log_instance.to_dict()
# create an instance of OwnerAuditLog from a dict
owner_audit_log_from_dict = OwnerAuditLog.from_dict(owner_audit_log_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


