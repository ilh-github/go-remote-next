# RecordingsApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**appendRecordingArtifact**](RecordingsApi.md#appendRecordingArtifact) | **PATCH** /v1/me/recordings/{recording_id}/artifact | Append one chunk of a resumable recording upload |
| [**createRecordingArtifactTicket**](RecordingsApi.md#createRecordingArtifactTicket) | **POST** /v1/me/recordings/{recording_id}/artifact-ticket | Mint a short-lived playback URL for a recording |
| [**createRecordingThumbnailTicket**](RecordingsApi.md#createRecordingThumbnailTicket) | **POST** /v1/me/recordings/{recording_id}/thumbnail-ticket | Mint a short-lived preview URL for a recording |
| [**createSessionRecording**](RecordingsApi.md#createSessionRecording) | **POST** /v1/me/recordings | Index a new recording |
| [**deleteSessionRecording**](RecordingsApi.md#deleteSessionRecording) | **DELETE** /v1/me/recordings/{recording_id} | Delete a recording and its stored media |
| [**downloadRecordingArtifact**](RecordingsApi.md#downloadRecordingArtifact) | **GET** /v1/me/recordings/{recording_id}/artifact | Download a recording&#39;s media bytes |
| [**downloadRecordingArtifactByTicket**](RecordingsApi.md#downloadRecordingArtifactByTicket) | **GET** /v1/recordings/{recording_id}/artifact | Fetch recording media with a playback ticket |
| [**downloadRecordingThumbnail**](RecordingsApi.md#downloadRecordingThumbnail) | **GET** /v1/me/recordings/{recording_id}/thumbnail | Download a recording&#39;s preview image |
| [**downloadRecordingThumbnailByTicket**](RecordingsApi.md#downloadRecordingThumbnailByTicket) | **GET** /v1/recordings/{recording_id}/thumbnail | Fetch a recording preview with a playback ticket |
| [**getSessionRecording**](RecordingsApi.md#getSessionRecording) | **GET** /v1/me/recordings/{recording_id} | One recording the caller owns |
| [**listSessionRecordings**](RecordingsApi.md#listSessionRecordings) | **GET** /v1/me/recordings | The caller&#39;s session-recording index |
| [**updateSessionRecording**](RecordingsApi.md#updateSessionRecording) | **PATCH** /v1/me/recordings/{recording_id} | Update a recording&#39;s lifecycle metadata |
| [**uploadRecordingArtifact**](RecordingsApi.md#uploadRecordingArtifact) | **POST** /v1/me/recordings/{recording_id}/artifact | Upload a recording&#39;s media bytes |
| [**uploadRecordingThumbnail**](RecordingsApi.md#uploadRecordingThumbnail) | **POST** /v1/me/recordings/{recording_id}/thumbnail | Upload a recording&#39;s preview image |


<a id="appendRecordingArtifact"></a>
# **appendRecordingArtifact**
> SessionRecording appendRecordingArtifact(ext, uploadOffset, recordingId, uploadLength, uploadComplete, uploadProbe, body)

Append one chunk of a resumable recording upload

The resumable half of the upload contract, for a capture that does not fit in one request body or a connection that drops halfway. The client sends chunks; the server answers with the offset it actually holds in the &#x60;Upload-Offset&#x60; response header, so a retry resumes from that number instead of starting over.  Three headers drive it. &#x60;Upload-Offset&#x60; (required) is where the chunk starts and must equal the bytes already staged; otherwise the answer is &#x60;409&#x60; with the authoritative offset in the same header and the client resyncs. &#x60;Upload-Length&#x60; (optional, &#x60;*&#x60; while unknown) declares the total size, which lets the server finish by itself when the last byte lands and lets it refuse a chunk that would overshoot. &#x60;Upload-Complete: 1&#x60; says this chunk ends the upload, which is how a client finishes without ever declaring a total.  A status probe asks where the upload stands without staging bytes. Send &#x60;Upload-Probe: 1&#x60; with a valid &#x60;Upload-Offset&#x60; and no body; the answer is &#x60;202&#x60; with the current offset, so a client that crashed can re-learn where it stopped without a second endpoint. An empty body with a valid offset does the same thing for raw-HTTP callers, but a generated client cannot send a zero-length body, which is why the explicit header exists.  The staging file (&#x60;&lt;artifact-path&gt;.part&#x60;) is subject to the same size limit as the single-shot upload, an empty artifact is refused on every path, and the recording lifecycle deletes a staging file that has been untouched for six hours. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = RecordingsApi()
val ext : kotlin.String = ext_example // kotlin.String | The artifact extension, allow-listed exactly like the raw-body POST shape.
val uploadOffset : kotlin.Long = 789 // kotlin.Long | The byte offset this chunk starts at.
val recordingId : kotlin.Long = 789 // kotlin.Long | The numeric id from a recording index row.
val uploadLength : kotlin.String = uploadLength_example // kotlin.String | The total size of the artifact, or `*` while it is unknown.
val uploadComplete : kotlin.String = uploadComplete_example // kotlin.String | Set to `1` when this chunk ends the upload.
val uploadProbe : kotlin.String = uploadProbe_example // kotlin.String | Set to `1` to ask for the current offset without staging any bytes. The generated clients use this instead of an empty body.
val body : java.io.File = BINARY_DATA_HERE // java.io.File | 
try {
    val result : SessionRecording = apiInstance.appendRecordingArtifact(ext, uploadOffset, recordingId, uploadLength, uploadComplete, uploadProbe, body)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling RecordingsApi#appendRecordingArtifact")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling RecordingsApi#appendRecordingArtifact")
    e.printStackTrace()
}
```

### Parameters
| **ext** | **kotlin.String**| The artifact extension, allow-listed exactly like the raw-body POST shape. | [enum: .mp4, .m4v, .mkv, .webm, .h264, .grf] |
| **uploadOffset** | **kotlin.Long**| The byte offset this chunk starts at. | |
| **recordingId** | **kotlin.Long**| The numeric id from a recording index row. | |
| **uploadLength** | **kotlin.String**| The total size of the artifact, or &#x60;*&#x60; while it is unknown. | [optional] |
| **uploadComplete** | **kotlin.String**| Set to &#x60;1&#x60; when this chunk ends the upload. | [optional] [enum: 0, 1] |
| **uploadProbe** | **kotlin.String**| Set to &#x60;1&#x60; to ask for the current offset without staging any bytes. The generated clients use this instead of an empty body. | [optional] [enum: 0, 1] |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **body** | **java.io.File**|  | [optional] |

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: application/octet-stream
 - **Accept**: application/json, application/problem+json

<a id="createRecordingArtifactTicket"></a>
# **createRecordingArtifactTicket**
> PlaybackTicket createRecordingArtifactTicket(recordingId)

Mint a short-lived playback URL for a recording

Owner-only. Returns a signed, time-limited &#x60;url&#x60; that can be fetched **without** an &#x60;Authorization&#x60; header, which is what makes playback work from &#x60;&lt;video src&gt;&#x60;. The ticket is bound to the recording, the owner and its purpose, so an artifact ticket cannot be replayed against the thumbnail route. Minting is stateless - nothing is written - and the link stops working when &#x60;expires_at&#x60; passes. &#x60;400&#x60; when playback tickets are disabled on the deployment (&#x60;RECORDING_URL_SECRET&#x60;/&#x60;API_KEY&#x60; unset); &#x60;404&#x60; when the recording has no stored artifact. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = RecordingsApi()
val recordingId : kotlin.Long = 789 // kotlin.Long | The numeric id from a recording index row.
try {
    val result : PlaybackTicket = apiInstance.createRecordingArtifactTicket(recordingId)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling RecordingsApi#createRecordingArtifactTicket")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling RecordingsApi#createRecordingArtifactTicket")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **recordingId** | **kotlin.Long**| The numeric id from a recording index row. | |

### Return type

[**PlaybackTicket**](PlaybackTicket.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

<a id="createRecordingThumbnailTicket"></a>
# **createRecordingThumbnailTicket**
> PlaybackTicket createRecordingThumbnailTicket(recordingId)

Mint a short-lived preview URL for a recording

Same rules as the artifact ticket, but the URL addresses the thumbnail route and the ticket&#39;s purpose is checked on redemption. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = RecordingsApi()
val recordingId : kotlin.Long = 789 // kotlin.Long | The numeric id from a recording index row.
try {
    val result : PlaybackTicket = apiInstance.createRecordingThumbnailTicket(recordingId)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling RecordingsApi#createRecordingThumbnailTicket")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling RecordingsApi#createRecordingThumbnailTicket")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **recordingId** | **kotlin.Long**| The numeric id from a recording index row. | |

### Return type

[**PlaybackTicket**](PlaybackTicket.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

<a id="createSessionRecording"></a>
# **createSessionRecording**
> SessionRecording createSessionRecording(sessionRecordingInput)

Index a new recording

Called by the capturing side when a session starts (or when a finished capture is handed over), so the owner sees the recording while it is still being written and the row can carry the outcome afterwards. &#x60;status&#x60; is one of &#x60;recording&#x60;, &#x60;completed&#x60; or &#x60;failed&#x60;; &#x60;started_at&#x60; defaults to now. The media is uploaded separately to &#x60;.../artifact&#x60;.  The row belongs to the *caller*: &#x60;device_id&#x60; is recorded as a label and is not checked for ownership, so a producer can index a capture for a machine that is not bound to it. Only the caller can then read or delete the row. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = RecordingsApi()
val sessionRecordingInput : SessionRecordingInput =  // SessionRecordingInput | 
try {
    val result : SessionRecording = apiInstance.createSessionRecording(sessionRecordingInput)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling RecordingsApi#createSessionRecording")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling RecordingsApi#createSessionRecording")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **sessionRecordingInput** | [**SessionRecordingInput**](SessionRecordingInput.md)|  | |

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

<a id="deleteSessionRecording"></a>
# **deleteSessionRecording**
> deleteSessionRecording(recordingId)

Delete a recording and its stored media

Removes the index row and the media files that belong to it, so the recording cannot be played back afterwards. There is no \&quot;already gone\&quot; success: a second delete is a &#x60;404&#x60;. The deletion is published on the owner&#39;s realtime stream and written to the audit trail as &#x60;recording.delete&#x60; with the actor&#39;s observed address. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = RecordingsApi()
val recordingId : kotlin.Long = 789 // kotlin.Long | The numeric id from a recording index row.
try {
    apiInstance.deleteSessionRecording(recordingId)
} catch (e: ClientException) {
    println("4xx response calling RecordingsApi#deleteSessionRecording")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling RecordingsApi#deleteSessionRecording")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **recordingId** | **kotlin.Long**| The numeric id from a recording index row. | |

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

<a id="downloadRecordingArtifact"></a>
# **downloadRecordingArtifact**
> java.io.File downloadRecordingArtifact(recordingId)

Download a recording&#39;s media bytes

Owner-only, &#x60;Authorization&#x60; (Bearer) required. The response is the stored file itself: the &#x60;Content-Type&#x60; follows the stored file&#39;s extension (&#x60;video/mp4&#x60; for &#x60;.mp4&#x60;, &#x60;application/octet-stream&#x60; otherwise) and &#x60;Content-Disposition&#x60; is &#x60;inline&#x60;. A browser cannot attach an &#x60;Authorization&#x60; header to &#x60;&lt;video src&gt;&#x60;, which is why the ticket routes below exist; this route is for API clients. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = RecordingsApi()
val recordingId : kotlin.Long = 789 // kotlin.Long | The numeric id from a recording index row.
try {
    val result : java.io.File = apiInstance.downloadRecordingArtifact(recordingId)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling RecordingsApi#downloadRecordingArtifact")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling RecordingsApi#downloadRecordingArtifact")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **recordingId** | **kotlin.Long**| The numeric id from a recording index row. | |

### Return type

[**java.io.File**](java.io.File.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/problem+json

<a id="downloadRecordingArtifactByTicket"></a>
# **downloadRecordingArtifactByTicket**
> java.io.File downloadRecordingArtifactByTicket(recordingId, t)

Fetch recording media with a playback ticket

The URL returned by &#x60;POST /v1/me/recordings/{recording_id}/artifact-ticket&#x60;. Deliberately registered **outside** the API-key and user-auth groups, because a browser &#x60;&lt;video&gt;&#x60; element can send neither header - the signed &#x60;t&#x60; query parameter is the authorization, and the ticket is bound to owner and purpose. &#x60;401&#x60; for a missing, malformed or expired ticket; &#x60;404&#x60; when the ticket is for the wrong route or the recording is gone. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = RecordingsApi()
val recordingId : kotlin.Long = 789 // kotlin.Long | The numeric id from a recording index row.
val t : kotlin.String = t_example // kotlin.String | The signed ticket from the artifact-ticket endpoint.
try {
    val result : java.io.File = apiInstance.downloadRecordingArtifactByTicket(recordingId, t)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling RecordingsApi#downloadRecordingArtifactByTicket")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling RecordingsApi#downloadRecordingArtifactByTicket")
    e.printStackTrace()
}
```

### Parameters
| **recordingId** | **kotlin.Long**| The numeric id from a recording index row. | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **t** | **kotlin.String**| The signed ticket from the artifact-ticket endpoint. | |

### Return type

[**java.io.File**](java.io.File.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/problem+json

<a id="downloadRecordingThumbnail"></a>
# **downloadRecordingThumbnail**
> java.io.File downloadRecordingThumbnail(recordingId)

Download a recording&#39;s preview image

Owner-only. Served with &#x60;Cache-Control: private, max-age&#x3D;300&#x60; because a preview is user-scoped and can change under the same URL. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = RecordingsApi()
val recordingId : kotlin.Long = 789 // kotlin.Long | The numeric id from a recording index row.
try {
    val result : java.io.File = apiInstance.downloadRecordingThumbnail(recordingId)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling RecordingsApi#downloadRecordingThumbnail")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling RecordingsApi#downloadRecordingThumbnail")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **recordingId** | **kotlin.Long**| The numeric id from a recording index row. | |

### Return type

[**java.io.File**](java.io.File.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/problem+json

<a id="downloadRecordingThumbnailByTicket"></a>
# **downloadRecordingThumbnailByTicket**
> java.io.File downloadRecordingThumbnailByTicket(recordingId, t)

Fetch a recording preview with a playback ticket

The URL returned by &#x60;POST /v1/me/recordings/{recording_id}/thumbnail-ticket&#x60;. Same ticket rules as the artifact route; an artifact ticket is rejected here (&#x60;404&#x60;) because the purpose is part of the signature. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = RecordingsApi()
val recordingId : kotlin.Long = 789 // kotlin.Long | The numeric id from a recording index row.
val t : kotlin.String = t_example // kotlin.String | The signed ticket from the thumbnail-ticket endpoint.
try {
    val result : java.io.File = apiInstance.downloadRecordingThumbnailByTicket(recordingId, t)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling RecordingsApi#downloadRecordingThumbnailByTicket")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling RecordingsApi#downloadRecordingThumbnailByTicket")
    e.printStackTrace()
}
```

### Parameters
| **recordingId** | **kotlin.Long**| The numeric id from a recording index row. | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **t** | **kotlin.String**| The signed ticket from the thumbnail-ticket endpoint. | |

### Return type

[**java.io.File**](java.io.File.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/problem+json

<a id="getSessionRecording"></a>
# **getSessionRecording**
> SessionRecording getSessionRecording(recordingId)

One recording the caller owns

The same row the list returns. Another user&#39;s id is a &#x60;404&#x60; rather than a &#x60;403&#x60;, so the route cannot be used to discover which recording ids exist. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = RecordingsApi()
val recordingId : kotlin.Long = 789 // kotlin.Long | The numeric id from a recording index row.
try {
    val result : SessionRecording = apiInstance.getSessionRecording(recordingId)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling RecordingsApi#getSessionRecording")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling RecordingsApi#getSessionRecording")
    e.printStackTrace()
}
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **recordingId** | **kotlin.Long**| The numeric id from a recording index row. | |

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

<a id="listSessionRecordings"></a>
# **listSessionRecordings**
> SessionRecordingList listSessionRecordings(cursor, limit)

The caller&#39;s session-recording index

Cursor-paginated, newest first. &#x60;next_cursor&#x60; is opaque: pass it back as &#x60;?cursor&#x3D;&#x60; to fetch the next page and stop when it is absent. The envelope here has no &#x60;total&#x60; because this endpoint does not count the whole set on every page, so an integrator must page until &#x60;next_cursor&#x60; is empty rather than trusting a count. This is the metadata index only; the media bytes live behind the per-recording &#x60;.../artifact&#x60; routes documented in &#x60;docs/user-api-docs.md&#x60;. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = RecordingsApi()
val cursor : kotlin.String = cursor_example // kotlin.String | 
val limit : kotlin.Int = 56 // kotlin.Int | 
try {
    val result : SessionRecordingList = apiInstance.listSessionRecordings(cursor, limit)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling RecordingsApi#listSessionRecordings")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling RecordingsApi#listSessionRecordings")
    e.printStackTrace()
}
```

### Parameters
| **cursor** | **kotlin.String**|  | [optional] |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **limit** | **kotlin.Int**|  | [optional] |

### Return type

[**SessionRecordingList**](SessionRecordingList.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

<a id="updateSessionRecording"></a>
# **updateSessionRecording**
> SessionRecording updateSessionRecording(recordingId, sessionRecordingPatch)

Update a recording&#39;s lifecycle metadata

Partial update: only the fields present in the body are written, so a producer can patch &#x60;status&#x60; when a capture starts and then &#x60;size_bytes&#x60;/&#x60;ended_at&#x60; when the upload finishes. &#x60;status&#x60; accepts the same three values as create; anything else is a &#x60;400&#x60; and writes nothing. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = RecordingsApi()
val recordingId : kotlin.Long = 789 // kotlin.Long | The numeric id from a recording index row.
val sessionRecordingPatch : SessionRecordingPatch =  // SessionRecordingPatch | 
try {
    val result : SessionRecording = apiInstance.updateSessionRecording(recordingId, sessionRecordingPatch)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling RecordingsApi#updateSessionRecording")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling RecordingsApi#updateSessionRecording")
    e.printStackTrace()
}
```

### Parameters
| **recordingId** | **kotlin.Long**| The numeric id from a recording index row. | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **sessionRecordingPatch** | [**SessionRecordingPatch**](SessionRecordingPatch.md)|  | |

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

<a id="uploadRecordingArtifact"></a>
# **uploadRecordingArtifact**
> SessionRecording uploadRecordingArtifact(recordingId, file, ext)

Upload a recording&#39;s media bytes

Two request shapes, because the producers differ: &#x60;multipart/form-data&#x60; with a &#x60;file&#x60; part (the browser and Qt clients) or the raw body with &#x60;?ext&#x3D;&#x60;. The extension is allow-listed (&#x60;.mp4&#x60;, &#x60;.m4v&#x60;, &#x60;.mkv&#x60;, &#x60;.webm&#x60;, &#x60;.h264&#x60;, &#x60;.grf&#x60;), the bytes are written to a temporary file and renamed into place so a dropped upload can never leave a half-written file under the real name, and the row is then updated with &#x60;artifact_path&#x60;, &#x60;artifact_url&#x60;, &#x60;format&#x60; and &#x60;size_bytes&#x60; in one step.  An empty body is a &#x60;400&#x60;; a body over the configured limit (&#x60;RECORDING_MAX_BYTES&#x60;, exposed as &#x60;store.max_bytes&#x60;) is a &#x60;400&#x60; as well, never a truncated file. Uploading again replaces the stored artifact. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = RecordingsApi()
val recordingId : kotlin.Long = 789 // kotlin.Long | The numeric id from a recording index row.
val file : java.io.File = BINARY_DATA_HERE // java.io.File | 
val ext : kotlin.String = ext_example // kotlin.String | Required for the raw-body shape; ignored for multipart.
try {
    val result : SessionRecording = apiInstance.uploadRecordingArtifact(recordingId, file, ext)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling RecordingsApi#uploadRecordingArtifact")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling RecordingsApi#uploadRecordingArtifact")
    e.printStackTrace()
}
```

### Parameters
| **recordingId** | **kotlin.Long**| The numeric id from a recording index row. | |
| **file** | **java.io.File**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **ext** | **kotlin.String**| Required for the raw-body shape; ignored for multipart. | [optional] [enum: .mp4, .m4v, .mkv, .webm, .h264, .grf] |

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: multipart/form-data
 - **Accept**: application/json, application/problem+json

<a id="uploadRecordingThumbnail"></a>
# **uploadRecordingThumbnail**
> SessionRecording uploadRecordingThumbnail(recordingId, file, ext)

Upload a recording&#39;s preview image

The same two shapes as the artifact upload (&#x60;multipart/form-data&#x60; with a &#x60;file&#x60; part, or the raw body with &#x60;?ext&#x3D;&#x60;), but the extension is limited to &#x60;.jpg&#x60;, &#x60;.jpeg&#x60; and &#x60;.png&#x60; and the size to 2 MiB - a preview is a small still, and an oversized one means the wrong file was sent.  The bytes are sniffed before they are stored rather than trusted from the name or the declared content type: the index renders this image inline, so a file that only claims to be a JPEG is a &#x60;400&#x60;. 

### Example
```kotlin
// Import classes:
//import io.quickdesk.signaling.infrastructure.*
//import io.quickdesk.signaling.models.*

val apiInstance = RecordingsApi()
val recordingId : kotlin.Long = 789 // kotlin.Long | The numeric id from a recording index row.
val file : java.io.File = BINARY_DATA_HERE // java.io.File | 
val ext : kotlin.String = ext_example // kotlin.String | Required for the raw-body shape; ignored for multipart.
try {
    val result : SessionRecording = apiInstance.uploadRecordingThumbnail(recordingId, file, ext)
    println(result)
} catch (e: ClientException) {
    println("4xx response calling RecordingsApi#uploadRecordingThumbnail")
    e.printStackTrace()
} catch (e: ServerException) {
    println("5xx response calling RecordingsApi#uploadRecordingThumbnail")
    e.printStackTrace()
}
```

### Parameters
| **recordingId** | **kotlin.Long**| The numeric id from a recording index row. | |
| **file** | **java.io.File**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **ext** | **kotlin.String**| Required for the raw-body shape; ignored for multipart. | [optional] [enum: .jpg, .jpeg, .png] |

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization


Configure ApiKeyAuth:
    ApiClient.apiKey["X-API-Key"] = ""
    ApiClient.apiKeyPrefix["X-API-Key"] = ""
Configure BearerAuth:
    ApiClient.accessToken = ""

### HTTP request headers

 - **Content-Type**: multipart/form-data
 - **Accept**: application/json, application/problem+json

