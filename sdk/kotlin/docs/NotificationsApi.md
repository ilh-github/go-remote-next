# NotificationsApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**getOfflineAlert**](NotificationsApi.md#getOfflineAlert) | **GET** /v1/me/devices/{device_id}/offline-alert | The owner&#39;s offline-alert rule for a device |
| [**listNotificationChannels**](NotificationsApi.md#listNotificationChannels) | **GET** /v1/me/notification-channels | The user&#39;s notification channels |
| [**putOfflineAlert**](NotificationsApi.md#putOfflineAlert) | **PUT** /v1/me/devices/{device_id}/offline-alert | Create or replace the owner&#39;s offline-alert rule |


<a id="getOfflineAlert"></a>
# **getOfflineAlert**
> OfflineAlertRule getOfflineAlert(deviceId)

The owner&#39;s offline-alert rule for a device

Returns the documented defaults (&#x60;enabled: false&#x60;, &#x60;after_minutes: 10&#x60;) when the owner never configured a rule, so a client can render the form without a second \&quot;does it exist\&quot; call. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = NotificationsApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
try {
    val result : OfflineAlertRule = apiInstance.getOfflineAlert(deviceId)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling NotificationsApi#getOfflineAlert")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling NotificationsApi#getOfflineAlert")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |

### Return type

[**OfflineAlertRule**](OfflineAlertRule.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

<a id="listNotificationChannels"></a>
# **listNotificationChannels**
> NotificationChannelList listNotificationChannels()

The user&#39;s notification channels

&#x60;supported_events&#x60; is the authoritative list of event types a channel may subscribe to. The signing secret is never returned; deliveries are signed with &#x60;signature_header&#x60; using &#x60;signature_scheme&#x60;. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = NotificationsApi()
try {
    val result : NotificationChannelList = apiInstance.listNotificationChannels()
    println(result)
} catch (e: ClientException) {
    println("4xx response calling NotificationsApi#listNotificationChannels")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling NotificationsApi#listNotificationChannels")
    e.printStackTrace()
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**NotificationChannelList**](NotificationChannelList.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

<a id="putOfflineAlert"></a>
# **putOfflineAlert**
> OfflineAlertRule putOfflineAlert(deviceId, offlineAlertInput)

Create or replace the owner&#39;s offline-alert rule

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = NotificationsApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
val offlineAlertInput : OfflineAlertInput =  // OfflineAlertInput | 
try {
    val result : OfflineAlertRule = apiInstance.putOfflineAlert(deviceId, offlineAlertInput)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling NotificationsApi#putOfflineAlert")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling NotificationsApi#putOfflineAlert")
    e.printStackTrace()
}
```

### Parameters
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **offlineAlertInput** | [**OfflineAlertInput**](OfflineAlertInput.md)|  | |

### Return type

[**OfflineAlertRule**](OfflineAlertRule.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

