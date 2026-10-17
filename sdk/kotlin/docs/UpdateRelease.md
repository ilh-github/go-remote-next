
# UpdateRelease

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **id** | **kotlin.Long** |  |  |
| **channel** | [**inline**](#Channel) |  |  |
| **platform** | [**inline**](#Platform) |  |  |
| **arch** | [**inline**](#Arch) |  |  |
| **version** | **kotlin.String** | Dotted numeric version, e.g. &#x60;4.2.5.0&#x60;. |  |
| **downloadUrl** | [**java.net.URI**](java.net.URI.md) | Absolute http(s) URL of the installer or archive. |  |
| **hostedHere** | **kotlin.Boolean** | True when this server stores and serves the bytes (so &#x60;download_url&#x60; points at &#x60;GET /v1/update-releases/{release_id}/artifact&#x60;), false when the installer lives on a host the operator manages. |  |
| **sha256** | **kotlin.String** | Lowercase hex SHA-256 of the artifact; clients verify what they downloaded. |  |
| **sizeBytes** | **kotlin.Long** |  |  |
| **mandatory** | **kotlin.Boolean** | When true, the update is not optional for older clients. |  |
| **publishedAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  |
| **notes** | **kotlin.String** |  |  [optional] |


<a id="Channel"></a>
## Enum: channel
| Name | Value |
| ---- | ----- |
| channel | stable, beta, nightly |


<a id="Platform"></a>
## Enum: platform
| Name | Value |
| ---- | ----- |
| platform | windows, macos, linux |


<a id="Arch"></a>
## Enum: arch
| Name | Value |
| ---- | ----- |
| arch | x64, arm64, universal |



