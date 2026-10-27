# DevicesApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**approveAccessApproval**](DevicesApi.md#approveAccessApproval) | **POST** /v1/me/devices/{device_id}/access-approvals/{approval_id}/approve | Approve a pending access request |
| [**banDeviceAccessSource**](DevicesApi.md#banDeviceAccessSource) | **POST** /v1/me/devices/{device_id}/access-bans | Permanently deny one source address |
| [**bindMyDevice**](DevicesApi.md#bindMyDevice) | **POST** /v1/me/devices | Bind a device to the authenticated user |
| [**claimAccessApproval**](DevicesApi.md#claimAccessApproval) | **POST** /v1/devices/{device_id}/access-approvals:claim | Exchange an approved request for a signal token |
| [**createAccessGrant**](DevicesApi.md#createAccessGrant) | **POST** /v1/me/devices/{device_id}/access-grants | Issue a time-boxed guest access code |
| [**denyAccessApproval**](DevicesApi.md#denyAccessApproval) | **POST** /v1/me/devices/{device_id}/access-approvals/{approval_id}/deny | Deny a pending access request |
| [**getAccessApprovalSummary**](DevicesApi.md#getAccessApprovalSummary) | **GET** /v1/me/devices/{device_id}/access-approvals:summary | Aggregated view of a device&#39;s approval queue |
| [**getAccessPolicy**](DevicesApi.md#getAccessPolicy) | **GET** /v1/me/devices/{device_id}/access-policy | The device&#39;s access policy |
| [**listAccessApprovals**](DevicesApi.md#listAccessApprovals) | **GET** /v1/me/devices/{device_id}/access-approvals | Second-factor approvals awaiting (or already decided by) the owner |
| [**listAccessGrants**](DevicesApi.md#listAccessGrants) | **GET** /v1/me/devices/{device_id}/access-grants | Guest access codes issued for a device |
| [**listMyDevices**](DevicesApi.md#listMyDevices) | **GET** /v1/me/devices | Devices owned by the authenticated user |
| [**putAccessPolicy**](DevicesApi.md#putAccessPolicy) | **PUT** /v1/me/devices/{device_id}/access-policy | Create or replace the device&#39;s access policy |
| [**reportHostMetrics**](DevicesApi.md#reportHostMetrics) | **POST** /v1/devices/{device_id}/metrics | Report the host&#39;s latest resource sample |
| [**revokeAccessGrant**](DevicesApi.md#revokeAccessGrant) | **DELETE** /v1/me/devices/{device_id}/access-grants/{grant_id} | Revoke a guest access code |
| [**unbanDeviceAccessSource**](DevicesApi.md#unbanDeviceAccessSource) | **POST** /v1/me/devices/{device_id}/access-bans:remove | Remove one permanent source ban |
| [**verifyAccessCode**](DevicesApi.md#verifyAccessCode) | **POST** /v1/devices/{device_id}/access-code:verify | Verify a device access code and mint a signal token |


<a id="approveAccessApproval"></a>
# **approveAccessApproval**
> AccessApproval approveAccessApproval(deviceId, approvalId, approveAccessApprovalRequest)

Approve a pending access request

Owner-only. The body is optional; a &#x60;reason&#x60; is stored on the row and shown to the requester. A request that was already decided or has expired returns &#x60;409&#x60; with the current status instead of pretending the decision landed. The conformance smoke pins the &#x60;404&#x60; shape with an unknown id; the happy path needs a real pending approval (an online device plus &#x60;approval_mode&#x60;), so it is covered by the end-to-end run in &#x60;docs/dev/自托管端到端联调记录.md&#x60; rather than the smoke. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DevicesApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
val approvalId : kotlin.Long = 789 // kotlin.Long | The numeric id from an approval row.
val approveAccessApprovalRequest : ApproveAccessApprovalRequest =  // ApproveAccessApprovalRequest | 
try {
    val result : AccessApproval = apiInstance.approveAccessApproval(deviceId, approvalId, approveAccessApprovalRequest)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DevicesApi#approveAccessApproval")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DevicesApi#approveAccessApproval")
    e.printStackTrace()
}
```

### Parameters
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |
| **approvalId** | **kotlin.Long**| The numeric id from an approval row. | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **approveAccessApprovalRequest** | [**ApproveAccessApprovalRequest**](ApproveAccessApprovalRequest.md)|  | [optional] |

### Return type

[**AccessApproval**](AccessApproval.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

<a id="banDeviceAccessSource"></a>
# **banDeviceAccessSource**
> DeviceAccessPolicy banDeviceAccessSource(deviceId, deviceAccessBanRequest)

Permanently deny one source address

Adds an exact IP or CIDR block to the device&#39;s persistent deny list and returns the complete updated policy. A deny match always wins over the allow list, time windows and a valid access code. Repeating the same source is a no-op; remove one source with &#x60;POST /v1/me/devices/{device_id}/access-bans:remove&#x60;. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DevicesApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
val deviceAccessBanRequest : DeviceAccessBanRequest =  // DeviceAccessBanRequest | 
try {
    val result : DeviceAccessPolicy = apiInstance.banDeviceAccessSource(deviceId, deviceAccessBanRequest)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DevicesApi#banDeviceAccessSource")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DevicesApi#banDeviceAccessSource")
    e.printStackTrace()
}
```

### Parameters
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **deviceAccessBanRequest** | [**DeviceAccessBanRequest**](DeviceAccessBanRequest.md)|  | |

### Return type

[**DeviceAccessPolicy**](DeviceAccessPolicy.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

<a id="bindMyDevice"></a>
# **bindMyDevice**
> Device bindMyDevice(bindMyDeviceRequest)

Bind a device to the authenticated user

Claims a provisioned device for the calling account, optionally taking it over from its previous owner. Idempotent: re-binding a device the caller already owns succeeds and is not counted against the account&#39;s device quota a second time.  Refused with &#x60;403 DEVICE_QUOTA_EXCEEDED&#x60; when the account is already at the &#x60;device_count&#x60; limit reported by &#x60;GET /v1/me&#x60;. A &#x60;device_count&#x60; of 0 means no limit is configured, which is the default for a new account. A takeover does not count against the new owner&#39;s quota, because the number of machines bound to that account does not change. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DevicesApi()
val bindMyDeviceRequest : BindMyDeviceRequest =  // BindMyDeviceRequest | 
try {
    val result : Device = apiInstance.bindMyDevice(bindMyDeviceRequest)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DevicesApi#bindMyDevice")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DevicesApi#bindMyDevice")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **bindMyDeviceRequest** | [**BindMyDeviceRequest**](BindMyDeviceRequest.md)|  | |

### Return type

[**Device**](Device.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

<a id="claimAccessApproval"></a>
# **claimAccessApproval**
> AccessCodeVerifyResponse claimAccessApproval(deviceId, approvalClaimRequest)

Exchange an approved request for a signal token

The requesting controller calls this with the &#x60;approval_id&#x60; and &#x60;approval_secret&#x60; it received from &#x60;access-code:verify&#x60;. It is API-key gated rather than user-authenticated because the caller has no account. &#x60;202&#x60; means the owner has not decided yet - keep polling with the same id and secret; &#x60;200&#x60; returns the signal token; &#x60;403&#x60; means denied or expired; &#x60;409&#x60; means the approval was already spent or the host went offline before the exchange. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DevicesApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
val approvalClaimRequest : ApprovalClaimRequest =  // ApprovalClaimRequest | 
try {
    val result : AccessCodeVerifyResponse = apiInstance.claimAccessApproval(deviceId, approvalClaimRequest)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DevicesApi#claimAccessApproval")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DevicesApi#claimAccessApproval")
    e.printStackTrace()
}
```

### Parameters
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **approvalClaimRequest** | [**ApprovalClaimRequest**](ApprovalClaimRequest.md)|  | |

### Return type

[**AccessCodeVerifyResponse**](AccessCodeVerifyResponse.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

<a id="createAccessGrant"></a>
# **createAccessGrant**
> AccessGrant createAccessGrant(deviceId, accessGrantInput)

Issue a time-boxed guest access code

The plaintext code is returned **once**, in the creation response; a grant is limited by expiry and uses, may be bound to a client id and/or source addresses, and is revoked with the DELETE below. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DevicesApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
val accessGrantInput : AccessGrantInput =  // AccessGrantInput | 
try {
    val result : AccessGrant = apiInstance.createAccessGrant(deviceId, accessGrantInput)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DevicesApi#createAccessGrant")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DevicesApi#createAccessGrant")
    e.printStackTrace()
}
```

### Parameters
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **accessGrantInput** | [**AccessGrantInput**](AccessGrantInput.md)|  | |

### Return type

[**AccessGrant**](AccessGrant.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

<a id="denyAccessApproval"></a>
# **denyAccessApproval**
> AccessApproval denyAccessApproval(deviceId, approvalId, approveAccessApprovalRequest)

Deny a pending access request

Owner-only, same shape as &#x60;approve&#x60;. Denying also arms the per-source denial cooldown (&#x60;APPROVAL_DENIAL_COOLDOWN_SECONDS&#x60;), so the same client cannot immediately re-request. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DevicesApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
val approvalId : kotlin.Long = 789 // kotlin.Long | The numeric id from an approval row.
val approveAccessApprovalRequest : ApproveAccessApprovalRequest =  // ApproveAccessApprovalRequest | 
try {
    val result : AccessApproval = apiInstance.denyAccessApproval(deviceId, approvalId, approveAccessApprovalRequest)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DevicesApi#denyAccessApproval")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DevicesApi#denyAccessApproval")
    e.printStackTrace()
}
```

### Parameters
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |
| **approvalId** | **kotlin.Long**| The numeric id from an approval row. | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **approveAccessApprovalRequest** | [**ApproveAccessApprovalRequest**](ApproveAccessApprovalRequest.md)|  | [optional] |

### Return type

[**AccessApproval**](AccessApproval.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

<a id="getAccessApprovalSummary"></a>
# **getAccessApprovalSummary**
> ApprovalSummary getAccessApprovalSummary(deviceId, since, limit)

Aggregated view of a device&#39;s approval queue

The row list shows requests one at a time; this is what makes \&quot;one address has been refused six times this afternoon\&quot; legible. The window is keyed on &#x60;created_at&#x60; for every count. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DevicesApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
val since : java.time.OffsetDateTime = 2013-10-20T19:20:30+01:00 // java.time.OffsetDateTime | RFC3339 start of the window; defaults to the last 24 hours.
val limit : kotlin.Int = 56 // kotlin.Int | How many refused sources to list (default 5, capped at 10).
try {
    val result : ApprovalSummary = apiInstance.getAccessApprovalSummary(deviceId, since, limit)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DevicesApi#getAccessApprovalSummary")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DevicesApi#getAccessApprovalSummary")
    e.printStackTrace()
}
```

### Parameters
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |
| **since** | **java.time.OffsetDateTime**| RFC3339 start of the window; defaults to the last 24 hours. | [optional] |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **limit** | **kotlin.Int**| How many refused sources to list (default 5, capped at 10). | [optional] |

### Return type

[**ApprovalSummary**](ApprovalSummary.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

<a id="getAccessPolicy"></a>
# **getAccessPolicy**
> DeviceAccessPolicy getAccessPolicy(deviceId)

The device&#39;s access policy

Returns the documented \&quot;no policy\&quot; defaults when the owner never saved one, so a client can render the form without a second existence check. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DevicesApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
try {
    val result : DeviceAccessPolicy = apiInstance.getAccessPolicy(deviceId)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DevicesApi#getAccessPolicy")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DevicesApi#getAccessPolicy")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |

### Return type

[**DeviceAccessPolicy**](DeviceAccessPolicy.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

<a id="listAccessApprovals"></a>
# **listAccessApprovals**
> ListAccessApprovals200Response listAccessApprovals(deviceId, status, limit)

Second-factor approvals awaiting (or already decided by) the owner

&#x60;status&#x60; is a comma-separated subset of &#x60;pending, approved, denied, expired, consumed&#x60; and defaults to &#x60;pending&#x60;. Approval is created by the device-side verify route when the policy asks for it; approving or denying happens on the sub-resource routes documented in &#x60;docs/user-api-docs.md&#x60; §5.3.5 (not yet part of this contract). 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DevicesApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
val status : kotlin.String = status_example // kotlin.String | 
val limit : kotlin.Int = 56 // kotlin.Int | 
try {
    val result : ListAccessApprovals200Response = apiInstance.listAccessApprovals(deviceId, status, limit)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DevicesApi#listAccessApprovals")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DevicesApi#listAccessApprovals")
    e.printStackTrace()
}
```

### Parameters
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |
| **status** | **kotlin.String**|  | [optional] |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **limit** | **kotlin.Int**|  | [optional] |

### Return type

[**ListAccessApprovals200Response**](ListAccessApprovals200Response.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

<a id="listAccessGrants"></a>
# **listAccessGrants**
> ListAccessGrants200Response listAccessGrants(deviceId)

Guest access codes issued for a device

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DevicesApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
try {
    val result : ListAccessGrants200Response = apiInstance.listAccessGrants(deviceId)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DevicesApi#listAccessGrants")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DevicesApi#listAccessGrants")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |

### Return type

[**ListAccessGrants200Response**](ListAccessGrants200Response.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

<a id="listMyDevices"></a>
# **listMyDevices**
> ListMyDevices200Response listMyDevices()

Devices owned by the authenticated user

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DevicesApi()
try {
    val result : ListMyDevices200Response = apiInstance.listMyDevices()
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DevicesApi#listMyDevices")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DevicesApi#listMyDevices")
    e.printStackTrace()
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**ListMyDevices200Response**](ListMyDevices200Response.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

<a id="putAccessPolicy"></a>
# **putAccessPolicy**
> DeviceAccessPolicy putAccessPolicy(deviceId, deviceAccessPolicyInput)

Create or replace the device&#39;s access policy

A **full replace**: every field not sent is reset to its zero value. The policy is enforced on the device-side verify route before the access code is compared, so a blocked client never gets to guess codes. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DevicesApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
val deviceAccessPolicyInput : DeviceAccessPolicyInput =  // DeviceAccessPolicyInput | 
try {
    val result : DeviceAccessPolicy = apiInstance.putAccessPolicy(deviceId, deviceAccessPolicyInput)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DevicesApi#putAccessPolicy")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DevicesApi#putAccessPolicy")
    e.printStackTrace()
}
```

### Parameters
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **deviceAccessPolicyInput** | [**DeviceAccessPolicyInput**](DeviceAccessPolicyInput.md)|  | |

### Return type

[**DeviceAccessPolicy**](DeviceAccessPolicy.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

<a id="reportHostMetrics"></a>
# **reportHostMetrics**
> ReportHostMetrics200Response reportHostMetrics(deviceId, hostMetricReport)

Report the host&#39;s latest resource sample

Host-only endpoint authenticated by both the deployment API key and the device secret. Values are percentages in the inclusive range 0..100. Repeating a report replaces the previous sample and advances the owner&#39;s metric-alert episode state. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DevicesApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
val hostMetricReport : HostMetricReport =  // HostMetricReport | 
try {
    val result : ReportHostMetrics200Response = apiInstance.reportHostMetrics(deviceId, hostMetricReport)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DevicesApi#reportHostMetrics")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DevicesApi#reportHostMetrics")
    e.printStackTrace()
}
```

### Parameters
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **hostMetricReport** | [**HostMetricReport**](HostMetricReport.md)|  | |

### Return type

[**ReportHostMetrics200Response**](ReportHostMetrics200Response.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

<a id="revokeAccessGrant"></a>
# **revokeAccessGrant**
> RevokeAccessGrant200Response revokeAccessGrant(deviceId, grantId)

Revoke a guest access code

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DevicesApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
val grantId : kotlin.Long = 789 // kotlin.Long | 
try {
    val result : RevokeAccessGrant200Response = apiInstance.revokeAccessGrant(deviceId, grantId)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DevicesApi#revokeAccessGrant")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DevicesApi#revokeAccessGrant")
    e.printStackTrace()
}
```

### Parameters
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **grantId** | **kotlin.Long**|  | |

### Return type

[**RevokeAccessGrant200Response**](RevokeAccessGrant200Response.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

<a id="unbanDeviceAccessSource"></a>
# **unbanDeviceAccessSource**
> DeviceAccessPolicy unbanDeviceAccessSource(deviceId, deviceAccessBanRequest)

Remove one permanent source ban

Removes the exact IP or CIDR entry from the device&#39;s persistent deny list and returns the complete updated policy. A source that is already absent, including on a device with no stored policy, is a successful no-op. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DevicesApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
val deviceAccessBanRequest : DeviceAccessBanRequest =  // DeviceAccessBanRequest | 
try {
    val result : DeviceAccessPolicy = apiInstance.unbanDeviceAccessSource(deviceId, deviceAccessBanRequest)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DevicesApi#unbanDeviceAccessSource")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DevicesApi#unbanDeviceAccessSource")
    e.printStackTrace()
}
```

### Parameters
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **deviceAccessBanRequest** | [**DeviceAccessBanRequest**](DeviceAccessBanRequest.md)|  | |

### Return type

[**DeviceAccessPolicy**](DeviceAccessPolicy.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

<a id="verifyAccessCode"></a>
# **verifyAccessCode**
> AccessCodeVerifyResponse verifyAccessCode(deviceId, accessCodeVerifyRequest)

Verify a device access code and mint a signal token

The controller-side entry point. On success it returns a signal token for the relay. When the device policy wants the owner to confirm the session it returns &#x60;202&#x60; with a pending approval instead, and the caller polls &#x60;/v1/devices/{device_id}/access-approvals:claim&#x60;. A rejected code is a &#x60;403&#x60; problem with &#x60;code: INVALID_CODE&#x60;.  This route is reachable without a user token (the controlling client is not logged in) but still requires the API key or an allowed Origin. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = DevicesApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
val accessCodeVerifyRequest : AccessCodeVerifyRequest =  // AccessCodeVerifyRequest | 
try {
    val result : AccessCodeVerifyResponse = apiInstance.verifyAccessCode(deviceId, accessCodeVerifyRequest)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling DevicesApi#verifyAccessCode")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling DevicesApi#verifyAccessCode")
    e.printStackTrace()
}
```

### Parameters
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **accessCodeVerifyRequest** | [**AccessCodeVerifyRequest**](AccessCodeVerifyRequest.md)|  | |

### Return type

[**AccessCodeVerifyResponse**](AccessCodeVerifyResponse.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

