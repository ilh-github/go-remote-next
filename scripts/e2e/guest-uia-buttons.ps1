# List every Button-ish element under the QuickDesk main window with its
# bounding rectangle, so unnamed QML buttons can still be targeted.
param([string]$WindowTitle = 'QuickDesk')

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName UIAutomationClient, UIAutomationTypes

$root = [System.Windows.Automation.AutomationElement]::RootElement
$target = $null
foreach ($process in @(Get-Process -Name QuickDesk -ErrorAction SilentlyContinue |
                       Where-Object { $_.MainWindowHandle -ne 0 })) {
    $cond = New-Object System.Windows.Automation.PropertyCondition(
        [System.Windows.Automation.AutomationElement]::ProcessIdProperty, $process.Id)
    $candidate = $root.FindFirst([System.Windows.Automation.TreeScope]::Children, $cond)
    if ($candidate) { $target = $candidate; break }
}
if (-not $target) { Write-Output 'no window'; exit 1 }

$all = $target.FindAll([System.Windows.Automation.TreeScope]::Descendants,
                       [System.Windows.Automation.Condition]::TrueCondition)
foreach ($element in $all) {
    $type = $element.Current.ControlType.ProgrammaticName
    if ($type -notmatch 'Button|ListItem|Image|Pane|Group') { continue }
    $rect = $element.Current.BoundingRectangle
    if ($rect.Width -le 0 -or $rect.Height -le 0) { continue }
    $cx = [int]($rect.X + $rect.Width / 2)
    $cy = [int]($rect.Y + $rect.Height / 2)
    Write-Output ("{0} center=({1},{2}) size={3}x{4} name='{5}' id='{6}'" -f `
        $type, $cx, $cy, [int]$rect.Width, [int]$rect.Height, `
        $element.Current.Name, $element.Current.AutomationId)
}
