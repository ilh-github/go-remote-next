# QuickDesk.Signaling.Client.Model.DeviceAccessPolicyInput
Full replace: fields not sent are reset to their zero value. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**RemoteDisabled** | **bool** | When true, no controller may start a session at all. | [optional] 
**AllowIps** | **List&lt;string&gt;** | Bare IPs or CIDR blocks; empty means \&quot;no IP restriction\&quot;. | [optional] 
**MaxSessions** | **int** | Concurrent controller cap; 0 means unlimited, negative is rejected. | [optional] 
**AllowedWindows** | [**List&lt;AccessWindow&gt;**](AccessWindow.md) | Recurring windows in which access is allowed; empty means always. | [optional] 
**Timezone** | **string** | IANA zone the windows are evaluated in; empty means the server&#39;s local zone. | [optional] 
**ExceptionDates** | [**List&lt;AccessExceptionDate&gt;**](AccessExceptionDate.md) | Per-date overrides (holidays, one-off closures, lunar dates). | [optional] 
**ApprovalRequired** | **bool** | Legacy switch kept for older clients; prefer &#x60;approval_mode&#x60;. | [optional] 
**ApprovalMode** | **string** | When the owner must approve a correct code before a session starts. | [optional] 
**ApprovalTrustedIps** | **List&lt;string&gt;** | Sources exempted by &#x60;approval_mode: untrusted_ip&#x60;. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

