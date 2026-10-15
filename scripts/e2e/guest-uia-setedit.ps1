# Set the value of the Nth Edit element inside a specific top-level window.
param(
    [Parameter(Mandatory = $true)][string]$Hwnd,
    [Parameter(Mandatory = $true)][int]$Index,
    [Parameter(Mandatory = $true)][string]$Value
)

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName UIAutomationClient, UIAutomationTypes

$target = [System.Windows.Automation.AutomationElement]::FromHandle([IntPtr][int64]$Hwnd)
if (-not $target) { Write-Output 'no window'; exit 1 }

$edits = New-Object System.Collections.ArrayList
foreach ($element in $target.FindAll([System.Windows.Automation.TreeScope]::Descendants,
                                     [System.Windows.Automation.Condition]::TrueCondition)) {
    if ($element.Current.ControlType -eq [System.Windows.Automation.ControlType]::Edit) {
        [void]$edits.Add($element)
    }
}
if ($Index -ge $edits.Count) {
    Write-Output "edit index $Index out of range (found $($edits.Count))"
    exit 1
}

$edit = $edits[$Index]
$edit.SetFocus()
Start-Sleep -Milliseconds 200
try {
    $pattern = $edit.GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern)
    $pattern.SetValue($Value)
} catch {
    Write-Output ("set failed: {0}" -f $_.Exception.Message)
    exit 2
}
Start-Sleep -Milliseconds 200
$check = $edit.GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern)
Write-Output ("setedit[{0}] = {1}" -f $Index, $check.Current.Value)
