# OwnerAuditLogPage

Cursor page of the caller's own data-channel audit rows, newest first. `next_cursor` is absent on the last page.  `details` is a JSON string rather than an object: it is stored verbatim as the write endpoint produced it, and parsing it as a string is what keeps the \"no clipboard text, ever\" property of that endpoint visible to a client instead of hiding it behind a typed field. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[OwnerAuditLog]**](OwnerAuditLog.md) |  | 
**next_cursor** | **str** |  | [optional] 
**total** | **int** |  | [optional] 

## Example

```python
from quickdesk_signaling.models.owner_audit_log_page import OwnerAuditLogPage

# TODO update the JSON string below
json = "{}"
# create an instance of OwnerAuditLogPage from a JSON string
owner_audit_log_page_instance = OwnerAuditLogPage.from_json(json)
# print the JSON string representation of the object
print(OwnerAuditLogPage.to_json())

# convert the object into a dict
owner_audit_log_page_dict = owner_audit_log_page_instance.to_dict()
# create an instance of OwnerAuditLogPage from a dict
owner_audit_log_page_from_dict = OwnerAuditLogPage.from_dict(owner_audit_log_page_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


