
# DeviceAccessPolicyInput

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
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
| **controlMode** | [**inline**](#ControlMode) | How several controllers share control of the machine. &#x60;shared&#x60; (the default) lets every connected controller drive it, which is what a device with no policy row already does. &#x60;exclusive&#x60; gives control to one controller at a time - a new one takes over and the previous holder is sent a &#x60;control_revoked&#x60; frame on its signaling socket. Separate from &#x60;max_sessions&#x60;, which caps how many people may watch. Omitted or empty means &#x60;shared&#x60;, because the endpoint replaces the whole policy. |  [optional] |


<a id="ApprovalMode"></a>
## Enum: approval_mode
| Name | Value |
| ---- | ----- |
| approvalMode | off, always, grant, untrusted_ip |


<a id="ControlMode"></a>
## Enum: control_mode
| Name | Value |
| ---- | ----- |
| controlMode | shared, exclusive |



