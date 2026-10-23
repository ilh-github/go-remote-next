# QuickDesk.Signaling.Client.Model.DeviceAccessPolicyInput
Full replace: fields not sent are reset to their zero value. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**RemoteDisabled** | **bool** | When true, no controller may start a session at all. | [optional] 
**AllowIps** | **List&lt;string&gt;** | Bare IPs or CIDR blocks; empty means \&quot;no IP restriction\&quot;. | [optional] 
**DenyIps** | **List&lt;string&gt;** | Bare IPs or CIDR blocks that are always refused; a match wins over &#x60;allow_ips&#x60;. | [optional] 
**MaxSessions** | **int** | Concurrent controller cap; 0 means unlimited, negative is rejected. | [optional] 
**AllowedWindows** | [**List&lt;AccessWindow&gt;**](AccessWindow.md) | Recurring windows in which access is allowed; empty means always. | [optional] 
**Timezone** | **string** | IANA zone the windows are evaluated in; empty means the server&#39;s local zone. | [optional] 
**ExceptionDates** | [**List&lt;AccessExceptionDate&gt;**](AccessExceptionDate.md) | Per-date overrides (holidays, one-off closures, lunar dates). | [optional] 
**ApprovalRequired** | **bool** | Legacy switch kept for older clients; prefer &#x60;approval_mode&#x60;. | [optional] 
**ApprovalMode** | **string** | When the owner must approve a correct code before a session starts. | [optional] 
**ApprovalTrustedIps** | **List&lt;string&gt;** | Sources exempted by &#x60;approval_mode: untrusted_ip&#x60;. | [optional] 
**ControlMode** | **string** | How several controllers share control of the machine. &#x60;shared&#x60; (the default) lets every connected controller drive it, which is what a device with no policy row already does. &#x60;exclusive&#x60; gives control to one controller at a time - a new one takes over and the previous holder is sent a &#x60;control_revoked&#x60; frame on its signaling socket. Separate from &#x60;max_sessions&#x60;, which caps how many people may watch. Omitted or empty means &#x60;shared&#x60;, because the endpoint replaces the whole policy. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

