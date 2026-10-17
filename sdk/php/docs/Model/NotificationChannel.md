# # NotificationChannel

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **int** |  |
**user_id** | **int** |  |
**name** | **string** |  |
**url** | **string** |  |
**events** | **string** | A JSON-encoded array of subscribed event types (a string, not an array, because that is how the column is stored). Decode it before matching against &#x60;supported_events&#x60;. |
**enabled** | **bool** |  |
**last_triggered** | **\DateTime** |  | [optional]
**last_status** | **int** | HTTP status of the most recent delivery attempt; 0 means none yet. |
**created_at** | **\DateTime** |  | [optional]
**updated_at** | **\DateTime** |  | [optional]

[[Back to Model list]](../../README.md#models) [[Back to API list]](../../README.md#endpoints) [[Back to README]](../../README.md)
