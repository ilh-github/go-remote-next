
# DeviceAccessPolicy

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **deviceId** | **kotlin.String** |  |  |
| **remoteDisabled** | **kotlin.Boolean** | When true, no controller may start a session at all. |  [optional] |
| **allowIps** | **kotlin.collections.List&lt;kotlin.String&gt;** | Bare IPs or CIDR blocks; empty means \&quot;no IP restriction\&quot;. |  [optional] |
| **denyIps** | **kotlin.collections.List&lt;kotlin.String&gt;** | Bare IPs or CIDR blocks that are always refused; a match wins over &#x60;allow_ips&#x60;. |  [optional] |
| **maxSessions** | **kotlin.Int** | Concurrent controller cap; 0 means unlimited, negative is rejected. |  [optional] |
| **allowedWindows** | [**kotlin.collections.List&lt;AccessWindow&gt;**](AccessWindow.md) | Recurring windows in which access is allowed; empty means always. |  [optional] |
| **timezone** | **kotlin.String** | IANA zone the windows are evaluated in; empty means the server&#39;s local zone. |  [optional] |
| **exceptionDates** | [**kotlin.collections.List&lt;AccessExceptionDate&gt;**](AccessExceptionDate.md) | Per-date overrides (holidays, one-off closures, lunar dates). |  [optional] |
| **approvalRequired** | **kotlin.Boolean** | Legacy switch kept for older clients; prefer &#x60;approval_mode&#x60;. |  [optional] |
| **approvalMode** | [**inline**](#ApprovalMode) | When the owner must approve a correct code before a session starts. |  [optional] |
| **approvalTrustedIps** | **kotlin.collections.List&lt;kotlin.String&gt;** | Sources exempted by &#x60;approval_mode: untrusted_ip&#x60;. |  [optional] |
| **updatedAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) | Omitted when the owner never saved a policy. |  [optional] |


<a id="ApprovalMode"></a>
## Enum: approval_mode
| Name | Value |
| ---- | ----- |
| approvalMode | off, always, grant, untrusted_ip |



