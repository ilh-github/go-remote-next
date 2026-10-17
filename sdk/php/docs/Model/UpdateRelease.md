# # UpdateRelease

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **int** |  |
**channel** | **string** |  |
**platform** | **string** |  |
**arch** | **string** |  |
**version** | **string** | Dotted numeric version, e.g. &#x60;4.2.5.0&#x60;. |
**download_url** | **string** | Absolute http(s) URL of the installer or archive. |
**hosted_here** | **bool** | True when this server stores and serves the bytes (so &#x60;download_url&#x60; points at &#x60;GET /v1/update-releases/{release_id}/artifact&#x60;), false when the installer lives on a host the operator manages. |
**sha256** | **string** | Lowercase hex SHA-256 of the artifact; clients verify what they downloaded. |
**size_bytes** | **int** |  |
**notes** | **string** |  | [optional]
**mandatory** | **bool** | When true, the update is not optional for older clients. |
**published_at** | **\DateTime** |  |

[[Back to Model list]](../../README.md#models) [[Back to API list]](../../README.md#endpoints) [[Back to README]](../../README.md)
