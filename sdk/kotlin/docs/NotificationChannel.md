
# NotificationChannel

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **id** | **kotlin.Long** |  |  |
| **userId** | **kotlin.Long** |  |  |
| **name** | **kotlin.String** |  |  |
| **url** | [**java.net.URI**](java.net.URI.md) |  |  |
| **events** | **kotlin.String** | A JSON-encoded array of subscribed event types (a string, not an array, because that is how the column is stored). Decode it before matching against &#x60;supported_events&#x60;.  |  |
| **enabled** | **kotlin.Boolean** |  |  |
| **lastStatus** | **kotlin.Int** | HTTP status of the most recent delivery attempt; 0 means none yet. |  |
| **lastTriggered** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  [optional] |
| **createdAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  [optional] |
| **updatedAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  [optional] |



