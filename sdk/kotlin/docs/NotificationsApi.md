# NotificationsApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**deleteMetricAlert**](NotificationsApi.md#deleteMetricAlert) | **DELETE** /v1/me/devices/{device_id}/metric-alert | Delete the owner&#39;s metric alert rule |
| [**getMetricAlert**](NotificationsApi.md#getMetricAlert) | **GET** /v1/me/devices/{device_id}/metric-alert | The owner&#39;s CPU, memory, and disk alert rule |
| [**getOfflineAlert**](NotificationsApi.md#getOfflineAlert) | **GET** /v1/me/devices/{device_id}/offline-alert | The owner&#39;s offline-alert rule for a device |
| [**listNotificationChannels**](NotificationsApi.md#listNotificationChannels) | **GET** /v1/me/notification-channels | The user&#39;s notification channels |
| [**putMetricAlert**](NotificationsApi.md#putMetricAlert) | **PUT** /v1/me/devices/{device_id}/metric-alert | Create or replace the owner&#39;s metric alert rule |
| [**putOfflineAlert**](NotificationsApi.md#putOfflineAlert) | **PUT** /v1/me/devices/{device_id}/offline-alert | Create or replace the owner&#39;s offline-alert rule |


<a id="deleteMetricAlert"></a>
# **deleteMetricAlert**
> deleteMetricAlert(deviceId)

Delete the owner&#39;s metric alert rule

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = NotificationsApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
try {
    apiInstance.deleteMetricAlert(deviceId)
} catch (e: ClientException) {
    println("4xx response calling NotificationsApi#deleteMetricAlert")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling NotificationsApi#deleteMetricAlert")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |

### Return type

null (empty response body)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/problem+json

<a id="getMetricAlert"></a>
# **getMetricAlert**
> MetricAlertRule getMetricAlert(deviceId)

The owner&#39;s CPU, memory, and disk alert rule

Returns safe defaults when the owner never configured a rule, along with the latest host sample and the independent episode state of each metric. Samples are one-per-device; this endpoint is not a time-series query. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = NotificationsApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
try {
    val result : MetricAlertRule = apiInstance.getMetricAlert(deviceId)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling NotificationsApi#getMetricAlert")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling NotificationsApi#getMetricAlert")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |

### Return type

[**MetricAlertRule**](MetricAlertRule.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

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

<a id="putMetricAlert"></a>
# **putMetricAlert**
> MetricAlertRule putMetricAlert(deviceId, metricAlertInput)

Create or replace the owner&#39;s metric alert rule

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = NotificationsApi()
val deviceId : kotlin.String = deviceId_example // kotlin.String | The device id shown on the controlled machine.
val metricAlertInput : MetricAlertInput =  // MetricAlertInput | 
try {
    val result : MetricAlertRule = apiInstance.putMetricAlert(deviceId, metricAlertInput)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling NotificationsApi#putMetricAlert")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling NotificationsApi#putMetricAlert")
    e.printStackTrace()
}
```

### Parameters
| **deviceId** | **kotlin.String**| The device id shown on the controlled machine. | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **metricAlertInput** | [**MetricAlertInput**](MetricAlertInput.md)|  | |

### Return type

[**MetricAlertRule**](MetricAlertRule.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: application/json
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

