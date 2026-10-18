# OpenAPIClient-php

This document is the contract for the operations a business system needs to
integrate with a self-hosted QuickDesk signaling server: authenticate,
enumerate the caller's devices, verify a device access code, and manage the
owner's access policy, guest access codes, second-factor approvals,
offline-alert rule and notification channels.

It deliberately covers a **subset**. The server exposes more (admin
endpoints, the approve/deny and claim halves of the approval flow,
recordings, realtime WebSockets); those are documented in
`docs/user-api-docs.md` and are added here as they are pinned by a
conformance test. Everything listed here is exercised against a live server
by `scripts/sdk/openapi-smoke.mjs`, so a change to a documented field fails
that check rather than silently invalidating this file.

That smoke is read-only by default; `QD_SMOKE_DEVICE=<device_id>` adds the
operations with side effects, and it restores what it touches: the
offline-alert rule and the access policy (including the `approval_mode` it
temporarily switches on to drive the approval flow) are put back exactly as
they were, and the recording it creates is deleted again.

Two residues are deliberate and are the reason this document says so: a
**revoked, never-used guest code** row from the create-then-revoke probe
(revocation, not deletion, is how a grant is disabled), and **two decided
approval rows** on the smoke device - one approved and consumed, one denied
- because a decision is history and the flow is auditable precisely by
leaving it.

Seven generated clients are committed - **TypeScript** under `sdk/ts`,
**Go** under `sdk/go`, **Python** under `sdk/python`, **Java** under
`sdk/java`, **C#** under `sdk/csharp`, **PHP** under `sdk/php` and
**Kotlin** under `sdk/kotlin`. All seven are kept in step with this file by
`scripts/sdk/check-generated-sdk.sh`, which CI runs: an edit here without a
regenerated client fails the build. Regenerate them with
`scripts/sdk/generate-client.sh sdk/ts`,
`scripts/sdk/generate-client.sh sdk/go go`,
`scripts/sdk/generate-client.sh sdk/python python`,
`scripts/sdk/generate-client.sh sdk/java java`,
`scripts/sdk/generate-client.sh sdk/csharp csharp`,
`scripts/sdk/generate-client.sh sdk/php php` and
`scripts/sdk/generate-client.sh sdk/kotlin kotlin` (a pinned
openapi-generator jar, so local and CI output are byte-identical); the
script takes the language as its last argument. CI compiles every client;
Go, Python, TypeScript and Java additionally run hand-written live smoke
examples against a running server. The `examples/` directories are excluded
from the freshness comparison, so they remain the only hand-written code
under `sdk/`.

All non-2xx responses are RFC 7807 problem details with
`Content-Type: application/problem+json`.



## Installation & Usage

### Requirements

PHP 8.1 and later.

### Composer

To install the bindings via [Composer](https://getcomposer.org/), add the following to `composer.json`:

```json
{
  "repositories": [
    {
      "type": "vcs",
      "url": "https://github.com/ilh-github/go-remote-next.git"
    }
  ],
  "require": {
    "ilh-github/go-remote-next": "*@dev"
  }
}
```

Then run `composer install`

### Manual Installation

Download the files and include `autoload.php`:

```php
<?php
require_once('/path/to/OpenAPIClient-php/vendor/autoload.php');
```

## Getting Started

Please follow the [installation procedure](#installation--usage) and then run the following:

```php
<?php
require_once(__DIR__ . '/vendor/autoload.php');



// Configure API key authorization: ApiKeyAuth
$config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setApiKey('X-API-Key', 'YOUR_API_KEY');
// Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
// $config = OpenAPI\Client\Configuration::getDefaultConfiguration()->setApiKeyPrefix('X-API-Key', 'Bearer');


$apiInstance = new OpenAPI\Client\Api\AuthApi(
    // If you want use custom http client, pass your client which implements `GuzzleHttp\ClientInterface`.
    // This is optional, `GuzzleHttp\Client` will be used as default.
    new GuzzleHttp\Client(),
    $config
);
$auth_session_request = new \OpenAPI\Client\Model\AuthSessionRequest(); // \OpenAPI\Client\Model\AuthSessionRequest

try {
    $result = $apiInstance->createSession($auth_session_request);
    print_r($result);
} catch (Exception $e) {
    echo 'Exception when calling AuthApi->createSession: ', $e->getMessage(), PHP_EOL;
}

```

## API Endpoints

All URIs are relative to *https://your-signaling-host*

Class | Method | HTTP request | Description
------------ | ------------- | ------------- | -------------
*AuthApi* | [**createSession**](docs/Api/AuthApi.md#createsession) | **POST** /v1/auth/sessions | Exchange a username/password for a session
*DeviceGroupsApi* | [**createMyDeviceGroup**](docs/Api/DeviceGroupsApi.md#createmydevicegroup) | **POST** /v1/me/device-groups | Create a private address-book group
*DeviceGroupsApi* | [**deleteMyDeviceGroup**](docs/Api/DeviceGroupsApi.md#deletemydevicegroup) | **DELETE** /v1/me/device-groups/{group_id} | Delete one of the caller&#39;s address-book groups
*DeviceGroupsApi* | [**listMyDeviceGroups**](docs/Api/DeviceGroupsApi.md#listmydevicegroups) | **GET** /v1/me/device-groups | The caller&#39;s private address-book groups
*DeviceGroupsApi* | [**replaceMyDeviceGroupDevices**](docs/Api/DeviceGroupsApi.md#replacemydevicegroupdevices) | **PUT** /v1/me/device-groups/{group_id}/devices | Replace the devices in one of the caller&#39;s groups
*DeviceGroupsApi* | [**updateMyDeviceGroup**](docs/Api/DeviceGroupsApi.md#updatemydevicegroup) | **PATCH** /v1/me/device-groups/{group_id} | Update one of the caller&#39;s address-book groups
*DevicesApi* | [**approveAccessApproval**](docs/Api/DevicesApi.md#approveaccessapproval) | **POST** /v1/me/devices/{device_id}/access-approvals/{approval_id}/approve | Approve a pending access request
*DevicesApi* | [**banDeviceAccessSource**](docs/Api/DevicesApi.md#bandeviceaccesssource) | **POST** /v1/me/devices/{device_id}/access-bans | Permanently deny one source address
*DevicesApi* | [**claimAccessApproval**](docs/Api/DevicesApi.md#claimaccessapproval) | **POST** /v1/devices/{device_id}/access-approvals:claim | Exchange an approved request for a signal token
*DevicesApi* | [**createAccessGrant**](docs/Api/DevicesApi.md#createaccessgrant) | **POST** /v1/me/devices/{device_id}/access-grants | Issue a time-boxed guest access code
*DevicesApi* | [**denyAccessApproval**](docs/Api/DevicesApi.md#denyaccessapproval) | **POST** /v1/me/devices/{device_id}/access-approvals/{approval_id}/deny | Deny a pending access request
*DevicesApi* | [**getAccessApprovalSummary**](docs/Api/DevicesApi.md#getaccessapprovalsummary) | **GET** /v1/me/devices/{device_id}/access-approvals:summary | Aggregated view of a device&#39;s approval queue
*DevicesApi* | [**getAccessPolicy**](docs/Api/DevicesApi.md#getaccesspolicy) | **GET** /v1/me/devices/{device_id}/access-policy | The device&#39;s access policy
*DevicesApi* | [**listAccessApprovals**](docs/Api/DevicesApi.md#listaccessapprovals) | **GET** /v1/me/devices/{device_id}/access-approvals | Second-factor approvals awaiting (or already decided by) the owner
*DevicesApi* | [**listAccessGrants**](docs/Api/DevicesApi.md#listaccessgrants) | **GET** /v1/me/devices/{device_id}/access-grants | Guest access codes issued for a device
*DevicesApi* | [**listMyDevices**](docs/Api/DevicesApi.md#listmydevices) | **GET** /v1/me/devices | Devices owned by the authenticated user
*DevicesApi* | [**putAccessPolicy**](docs/Api/DevicesApi.md#putaccesspolicy) | **PUT** /v1/me/devices/{device_id}/access-policy | Create or replace the device&#39;s access policy
*DevicesApi* | [**reportHostMetrics**](docs/Api/DevicesApi.md#reporthostmetrics) | **POST** /v1/devices/{device_id}/metrics | Report the host&#39;s latest resource sample
*DevicesApi* | [**revokeAccessGrant**](docs/Api/DevicesApi.md#revokeaccessgrant) | **DELETE** /v1/me/devices/{device_id}/access-grants/{grant_id} | Revoke a guest access code
*DevicesApi* | [**unbanDeviceAccessSource**](docs/Api/DevicesApi.md#unbandeviceaccesssource) | **POST** /v1/me/devices/{device_id}/access-bans:remove | Remove one permanent source ban
*DevicesApi* | [**verifyAccessCode**](docs/Api/DevicesApi.md#verifyaccesscode) | **POST** /v1/devices/{device_id}/access-code:verify | Verify a device access code and mint a signal token
*FileTransfersApi* | [**createFileTransferAudit**](docs/Api/FileTransfersApi.md#createfiletransferaudit) | **POST** /v1/me/file-transfer-audits | Report a controller-observed file transfer for auditing
*MeApi* | [**getMe**](docs/Api/MeApi.md#getme) | **GET** /v1/me | The authenticated user
*NotificationsApi* | [**deleteMetricAlert**](docs/Api/NotificationsApi.md#deletemetricalert) | **DELETE** /v1/me/devices/{device_id}/metric-alert | Delete the owner&#39;s metric alert rule
*NotificationsApi* | [**getMetricAlert**](docs/Api/NotificationsApi.md#getmetricalert) | **GET** /v1/me/devices/{device_id}/metric-alert | The owner&#39;s CPU, memory, and disk alert rule
*NotificationsApi* | [**getOfflineAlert**](docs/Api/NotificationsApi.md#getofflinealert) | **GET** /v1/me/devices/{device_id}/offline-alert | The owner&#39;s offline-alert rule for a device
*NotificationsApi* | [**listNotificationChannels**](docs/Api/NotificationsApi.md#listnotificationchannels) | **GET** /v1/me/notification-channels | The user&#39;s notification channels
*NotificationsApi* | [**putMetricAlert**](docs/Api/NotificationsApi.md#putmetricalert) | **PUT** /v1/me/devices/{device_id}/metric-alert | Create or replace the owner&#39;s metric alert rule
*NotificationsApi* | [**putOfflineAlert**](docs/Api/NotificationsApi.md#putofflinealert) | **PUT** /v1/me/devices/{device_id}/offline-alert | Create or replace the owner&#39;s offline-alert rule
*RecordingsApi* | [**appendRecordingArtifact**](docs/Api/RecordingsApi.md#appendrecordingartifact) | **PATCH** /v1/me/recordings/{recording_id}/artifact | Append one chunk of a resumable recording upload
*RecordingsApi* | [**createRecordingArtifactTicket**](docs/Api/RecordingsApi.md#createrecordingartifactticket) | **POST** /v1/me/recordings/{recording_id}/artifact-ticket | Mint a short-lived playback URL for a recording
*RecordingsApi* | [**createRecordingThumbnailTicket**](docs/Api/RecordingsApi.md#createrecordingthumbnailticket) | **POST** /v1/me/recordings/{recording_id}/thumbnail-ticket | Mint a short-lived preview URL for a recording
*RecordingsApi* | [**createSessionRecording**](docs/Api/RecordingsApi.md#createsessionrecording) | **POST** /v1/me/recordings | Index a new recording
*RecordingsApi* | [**deleteSessionRecording**](docs/Api/RecordingsApi.md#deletesessionrecording) | **DELETE** /v1/me/recordings/{recording_id} | Delete a recording and its stored media
*RecordingsApi* | [**downloadRecordingArtifact**](docs/Api/RecordingsApi.md#downloadrecordingartifact) | **GET** /v1/me/recordings/{recording_id}/artifact | Download a recording&#39;s media bytes
*RecordingsApi* | [**downloadRecordingArtifactByTicket**](docs/Api/RecordingsApi.md#downloadrecordingartifactbyticket) | **GET** /v1/recordings/{recording_id}/artifact | Fetch recording media with a playback ticket
*RecordingsApi* | [**downloadRecordingThumbnail**](docs/Api/RecordingsApi.md#downloadrecordingthumbnail) | **GET** /v1/me/recordings/{recording_id}/thumbnail | Download a recording&#39;s preview image
*RecordingsApi* | [**downloadRecordingThumbnailByTicket**](docs/Api/RecordingsApi.md#downloadrecordingthumbnailbyticket) | **GET** /v1/recordings/{recording_id}/thumbnail | Fetch a recording preview with a playback ticket
*RecordingsApi* | [**getSessionRecording**](docs/Api/RecordingsApi.md#getsessionrecording) | **GET** /v1/me/recordings/{recording_id} | One recording the caller owns
*RecordingsApi* | [**listSessionRecordings**](docs/Api/RecordingsApi.md#listsessionrecordings) | **GET** /v1/me/recordings | The caller&#39;s session-recording index
*RecordingsApi* | [**updateSessionRecording**](docs/Api/RecordingsApi.md#updatesessionrecording) | **PATCH** /v1/me/recordings/{recording_id} | Update a recording&#39;s lifecycle metadata
*RecordingsApi* | [**uploadRecordingArtifact**](docs/Api/RecordingsApi.md#uploadrecordingartifact) | **POST** /v1/me/recordings/{recording_id}/artifact | Upload a recording&#39;s media bytes
*RecordingsApi* | [**uploadRecordingThumbnail**](docs/Api/RecordingsApi.md#uploadrecordingthumbnail) | **POST** /v1/me/recordings/{recording_id}/thumbnail | Upload a recording&#39;s preview image
*RemoteSessionsApi* | [**getRemoteSession**](docs/Api/RemoteSessionsApi.md#getremotesession) | **GET** /v1/me/remote-sessions/{session_id} | One control session with the records that share its id
*RemoteSessionsApi* | [**listDeviceRemoteSessions**](docs/Api/RemoteSessionsApi.md#listdeviceremotesessions) | **GET** /v1/me/devices/{device_id}/sessions | One device&#39;s control sessions, each with its related records
*RemoteSessionsApi* | [**listRemoteSessions**](docs/Api/RemoteSessionsApi.md#listremotesessions) | **GET** /v1/me/remote-sessions | The caller&#39;s control sessions, open and finished
*RemoteSessionsApi* | [**revokeDeviceSessions**](docs/Api/RemoteSessionsApi.md#revokedevicesessions) | **POST** /v1/me/devices/{device_id}/sessions:revoke | End every live control session on one device
*RemoteSessionsApi* | [**revokeRemoteSession**](docs/Api/RemoteSessionsApi.md#revokeremotesession) | **POST** /v1/me/remote-sessions/{session_id}/revoke | End one of the caller&#39;s control sessions
*UpdateApi* | [**checkForUpdate**](docs/Api/UpdateApi.md#checkforupdate) | **GET** /v1/update:check | Check the self-hosted update channel for a newer build
*UpdateApi* | [**downloadUpdateArtifact**](docs/Api/UpdateApi.md#downloadupdateartifact) | **GET** /v1/update-releases/{release_id}/artifact | Download an installer this server hosts itself

## Models

- [AccessApproval](docs/Model/AccessApproval.md)
- [AccessCodeApprovalPending](docs/Model/AccessCodeApprovalPending.md)
- [AccessCodeVerifyRequest](docs/Model/AccessCodeVerifyRequest.md)
- [AccessCodeVerifyResponse](docs/Model/AccessCodeVerifyResponse.md)
- [AccessExceptionDate](docs/Model/AccessExceptionDate.md)
- [AccessGrant](docs/Model/AccessGrant.md)
- [AccessGrantInput](docs/Model/AccessGrantInput.md)
- [AccessWindow](docs/Model/AccessWindow.md)
- [ApprovalAlreadyDecided](docs/Model/ApprovalAlreadyDecided.md)
- [ApprovalClaimConsumed](docs/Model/ApprovalClaimConsumed.md)
- [ApprovalClaimPending](docs/Model/ApprovalClaimPending.md)
- [ApprovalClaimRequest](docs/Model/ApprovalClaimRequest.md)
- [ApprovalSummary](docs/Model/ApprovalSummary.md)
- [ApprovalSummaryByStatus](docs/Model/ApprovalSummaryByStatus.md)
- [ApprovalSummaryDeniedSourcesInner](docs/Model/ApprovalSummaryDeniedSourcesInner.md)
- [ApproveAccessApprovalRequest](docs/Model/ApproveAccessApprovalRequest.md)
- [AuthSessionRequest](docs/Model/AuthSessionRequest.md)
- [AuthSessionResponse](docs/Model/AuthSessionResponse.md)
- [DeleteMyDeviceGroup200Response](docs/Model/DeleteMyDeviceGroup200Response.md)
- [Device](docs/Model/Device.md)
- [DeviceAccessBanRequest](docs/Model/DeviceAccessBanRequest.md)
- [DeviceAccessPolicy](docs/Model/DeviceAccessPolicy.md)
- [DeviceAccessPolicyInput](docs/Model/DeviceAccessPolicyInput.md)
- [FileTransferAuditInput](docs/Model/FileTransferAuditInput.md)
- [HostMetricReport](docs/Model/HostMetricReport.md)
- [ListAccessApprovals200Response](docs/Model/ListAccessApprovals200Response.md)
- [ListAccessGrants200Response](docs/Model/ListAccessGrants200Response.md)
- [ListMyDevices200Response](docs/Model/ListMyDevices200Response.md)
- [MaintenanceWindow](docs/Model/MaintenanceWindow.md)
- [MetricAlertInput](docs/Model/MetricAlertInput.md)
- [MetricAlertRule](docs/Model/MetricAlertRule.md)
- [MetricAlertState](docs/Model/MetricAlertState.md)
- [MetricAlertStates](docs/Model/MetricAlertStates.md)
- [MetricSnapshot](docs/Model/MetricSnapshot.md)
- [NotificationChannel](docs/Model/NotificationChannel.md)
- [NotificationChannelList](docs/Model/NotificationChannelList.md)
- [OfflineAlertInput](docs/Model/OfflineAlertInput.md)
- [OfflineAlertRule](docs/Model/OfflineAlertRule.md)
- [PlaybackTicket](docs/Model/PlaybackTicket.md)
- [Problem](docs/Model/Problem.md)
- [RemoteSession](docs/Model/RemoteSession.md)
- [RemoteSessionAccessLog](docs/Model/RemoteSessionAccessLog.md)
- [RemoteSessionBatchRevocation](docs/Model/RemoteSessionBatchRevocation.md)
- [RemoteSessionBatchRevocationSessionsInner](docs/Model/RemoteSessionBatchRevocationSessionsInner.md)
- [RemoteSessionDetail](docs/Model/RemoteSessionDetail.md)
- [RemoteSessionDetailCounts](docs/Model/RemoteSessionDetailCounts.md)
- [RemoteSessionDetailList](docs/Model/RemoteSessionDetailList.md)
- [RemoteSessionList](docs/Model/RemoteSessionList.md)
- [RemoteSessionRevocation](docs/Model/RemoteSessionRevocation.md)
- [ReportHostMetrics200Response](docs/Model/ReportHostMetrics200Response.md)
- [RevokeAccessGrant200Response](docs/Model/RevokeAccessGrant200Response.md)
- [SessionRecording](docs/Model/SessionRecording.md)
- [SessionRecordingInput](docs/Model/SessionRecordingInput.md)
- [SessionRecordingList](docs/Model/SessionRecordingList.md)
- [SessionRecordingPatch](docs/Model/SessionRecordingPatch.md)
- [UpdateCheckResult](docs/Model/UpdateCheckResult.md)
- [UpdateRelease](docs/Model/UpdateRelease.md)
- [User](docs/Model/User.md)
- [UserDeviceGroup](docs/Model/UserDeviceGroup.md)
- [UserDeviceGroupDevicesInput](docs/Model/UserDeviceGroupDevicesInput.md)
- [UserDeviceGroupInput](docs/Model/UserDeviceGroupInput.md)
- [UserDeviceGroupList](docs/Model/UserDeviceGroupList.md)
- [UserDeviceGroupPatch](docs/Model/UserDeviceGroupPatch.md)

## Authorization

Authentication schemes defined for the API:
### ApiKeyAuth

- **Type**: API key
- **API key parameter name**: X-API-Key
- **Location**: HTTP header


### BearerAuth

- **Type**: Bearer authentication (opaque)

## Tests

To run the tests, use:

```bash
composer install
vendor/bin/phpunit
```

## Author



## About this package

This PHP package is automatically generated by the [OpenAPI Generator](https://openapi-generator.tech) project:

- API version: `1.0.0`
    - Generator version: `7.14.0`
- Build package: `org.openapitools.codegen.languages.PhpClientCodegen`
