# # ApprovalSummary

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**since** | **\DateTime** | Inclusive start of the window the counts cover. |
**total** | **int** | Every request created inside the window, whatever its outcome. |
**by_status** | [**\OpenAPI\Client\Model\ApprovalSummaryByStatus**](ApprovalSummaryByStatus.md) |  |
**denied_sources** | [**\OpenAPI\Client\Model\ApprovalSummaryDeniedSourcesInner[]**](ApprovalSummaryDeniedSourcesInner.md) | Refused sources inside the window, most refusals first. |
**denial_cooldown_seconds** | **int** | The live APPROVAL_DENIAL_COOLDOWN_SECONDS, so an owner can tell \&quot;refused right now\&quot; from \&quot;refused a while ago\&quot;. |
**denial_cooldown_multiplier** | **int** | The live APPROVAL_DENIAL_COOLDOWN_MULTIPLIER; 1 means each further consecutive refusal costs the same as the last. |
**denial_cooldown_max_seconds** | **int** | The live APPROVAL_DENIAL_COOLDOWN_MAX_SECONDS, the ceiling the escalation cannot pass; 0 means uncapped. |

[[Back to Model list]](../../README.md#models) [[Back to API list]](../../README.md#endpoints) [[Back to README]](../../README.md)
