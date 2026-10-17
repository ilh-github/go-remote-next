# QuickDesk.Signaling.Client.Model.ApprovalSummary

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Since** | **DateTime** | Inclusive start of the window the counts cover. | 
**Total** | **long** | Every request created inside the window, whatever its outcome. | 
**ByStatus** | [**ApprovalSummaryByStatus**](ApprovalSummaryByStatus.md) |  | 
**DeniedSources** | [**List&lt;ApprovalSummaryDeniedSourcesInner&gt;**](ApprovalSummaryDeniedSourcesInner.md) | Refused sources inside the window, most refusals first. | 
**DenialCooldownSeconds** | **int** | The live APPROVAL_DENIAL_COOLDOWN_SECONDS, so an owner can tell \&quot;refused right now\&quot; from \&quot;refused a while ago\&quot;. | 
**DenialCooldownMultiplier** | **int** | The live APPROVAL_DENIAL_COOLDOWN_MULTIPLIER; 1 means each further consecutive refusal costs the same as the last. | 
**DenialCooldownMaxSeconds** | **int** | The live APPROVAL_DENIAL_COOLDOWN_MAX_SECONDS, the ceiling the escalation cannot pass; 0 means uncapped. | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

