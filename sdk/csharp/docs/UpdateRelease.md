# QuickDesk.Signaling.Client.Model.UpdateRelease
One build published to the self-hosted update channel. The artifact itself is always fetchable at `download_url`: either an absolute URL on a host the operator controls, or this server's own `GET /v1/update-releases/{release_id}/artifact` when the installer was uploaded here (`hosted_here` is true). A release with no fetchable bytes is a placeholder and is never returned to a client. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **long** |  | 
**Channel** | **string** |  | 
**Platform** | **string** |  | 
**Arch** | **string** |  | 
**VarVersion** | **string** | Dotted numeric version, e.g. &#x60;4.2.5.0&#x60;. | 
**DownloadUrl** | **string** | Absolute http(s) URL of the installer or archive. | 
**HostedHere** | **bool** | True when this server stores and serves the bytes (so &#x60;download_url&#x60; points at &#x60;GET /v1/update-releases/{release_id}/artifact&#x60;), false when the installer lives on a host the operator manages. | 
**Sha256** | **string** | Lowercase hex SHA-256 of the artifact; clients verify what they downloaded. | 
**SizeBytes** | **long** |  | 
**Notes** | **string** |  | [optional] 
**Mandatory** | **bool** | When true, the update is not optional for older clients. | 
**PublishedAt** | **DateTime** |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

