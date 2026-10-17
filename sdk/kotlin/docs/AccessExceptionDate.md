
# AccessExceptionDate

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **mode** | [**inline**](#Mode) |  |  |
| **date** | **kotlin.String** | First date of the exception (YYYY-MM-DD). |  [optional] |
| **endDate** | **kotlin.String** | Last date of a range (YYYY-MM-DD); omit for a single day. |  [optional] |
| **repeat** | [**inline**](#Repeat) |  |  [optional] |
| **holiday** | **kotlin.String** | A statutory holiday name from the server&#39;s calendar, e.g. 春节. |  [optional] |
| **holidayDay** | [**inline**](#HolidayDay) | Which dates of the holiday an entry covers (default &#x60;off&#x60;). |  [optional] |
| **lunarMonth** | **kotlin.Int** |  |  [optional] |
| **lunarDay** | **kotlin.Int** |  |  [optional] |


<a id="Mode"></a>
## Enum: mode
| Name | Value |
| ---- | ----- |
| mode | allow, deny |


<a id="Repeat"></a>
## Enum: repeat
| Name | Value |
| ---- | ----- |
| repeat | yearly, lunar |


<a id="HolidayDay"></a>
## Enum: holiday_day
| Name | Value |
| ---- | ----- |
| holidayDay | off, work |



