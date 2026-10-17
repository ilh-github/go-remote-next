# # AccessCodeVerifyResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**status** | **string** | Which shape this response is. |
**signal_token** | **string** | Present when &#x60;status&#x60; is &#x60;granted&#x60;. | [optional]
**expires_at** | **\DateTime** | When the signal token (granted) or the approval (pending) expires. | [optional]
**approval_id** | **int** | Present when &#x60;status&#x60; is &#x60;pending&#x60;. | [optional]
**approval_secret** | **string** | Present when &#x60;status&#x60; is &#x60;pending&#x60;. Shown once; exchanged for a signal token at the claim endpoint. | [optional]
**poll_after_ms** | **int** | Present when &#x60;status&#x60; is &#x60;pending&#x60;. | [optional]

[[Back to Model list]](../../README.md#models) [[Back to API list]](../../README.md#endpoints) [[Back to README]](../../README.md)
