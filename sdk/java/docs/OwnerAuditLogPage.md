

# OwnerAuditLogPage

Cursor page of the caller's own data-channel audit rows, newest first. `next_cursor` is absent on the last page.  `details` is a JSON string rather than an object: it is stored verbatim as the write endpoint produced it, and parsing it as a string is what keeps the \"no clipboard text, ever\" property of that endpoint visible to a client instead of hiding it behind a typed field. 

## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**items** | [**List&lt;OwnerAuditLog&gt;**](OwnerAuditLog.md) |  |  |
|**nextCursor** | **String** |  |  [optional] |
|**total** | **Long** |  |  [optional] |



