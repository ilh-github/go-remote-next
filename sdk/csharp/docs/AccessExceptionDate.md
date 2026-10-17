# QuickDesk.Signaling.Client.Model.AccessExceptionDate
Exactly one of `date` (+ optional `end_date`), `holiday`, or `repeat: \"lunar\"` with the lunar month/day. `mode` is `allow` or `deny`; an allow and a deny entry covering the same day are rejected. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Date** | **string** | First date of the exception (YYYY-MM-DD). | [optional] 
**EndDate** | **string** | Last date of a range (YYYY-MM-DD); omit for a single day. | [optional] 
**Mode** | **string** |  | 
**Repeat** | **string** |  | [optional] 
**Holiday** | **string** | A statutory holiday name from the server&#39;s calendar, e.g. 春节. | [optional] 
**HolidayDay** | **string** | Which dates of the holiday an entry covers (default &#x60;off&#x60;). | [optional] 
**LunarMonth** | **int** |  | [optional] 
**LunarDay** | **int** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

