# Invoke a QuickDesk UI element through UI Automation instead of synthetic
# mouse input. Matches the first descendant whose AutomationId contains the
# given suffix.
param(
    [string]$AutomationIdSuffix = '',
    [string]$Name = '',
    [string]$Hwnd = ''
)

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName UIAutomationClient, UIAutomationTypes

$root = [System.Windows.Automation.AutomationElement]::RootElement
$target = $null
if ($Hwnd) {
    $target = [System.Windows.Automation.AutomationElement]::FromHandle([IntPtr][int64]$Hwnd)
}
if (-not $target) {
    foreach ($process in @(Get-Process -Name QuickDesk -ErrorAction SilentlyContinue |
                           Where-Object { $_.MainWindowHandle -ne 0 })) {
        $cond = New-Object System.Windows.Automation.PropertyCondition(
            [System.Windows.Automation.AutomationElement]::ProcessIdProperty, $process.Id)
        $candidate = $root.FindFirst([System.Windows.Automation.TreeScope]::Children, $cond)
        if ($candidate) { $target = $candidate; break }
    }
}
if (-not $target) { Write-Output 'no window'; exit 1 }

$all = $target.FindAll([System.Windows.Automation.TreeScope]::Descendants,
                       [System.Windows.Automation.Condition]::TrueCondition)
foreach ($element in $all) {
    $id = $element.Current.AutomationId
    $elementName = $element.Current.Name
    $idHit = $AutomationIdSuffix -and $id -like "*$AutomationIdSuffix*"
    $nameHit = $Name -and $elementName -eq $Name
    if (-not ($idHit -or $nameHit)) { continue }
    $rect = $element.Current.BoundingRectangle
    Write-Output ("match id='{0}' name='{1}' center=({2},{3})" -f $id, $elementName,
        [int]($rect.X + $rect.Width / 2), [int]($rect.Y + $rect.Height / 2))
    try {
        $pattern = $element.GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern)
        $pattern.Invoke()
        Write-Output 'invoked'
    } catch {
        Write-Output ("invoke failed: {0}" -f $_.Exception.Message)
        exit 2
    }
    exit 0
}

Write-Output "no element matching id='$AutomationIdSuffix' name='$Name'"
exit 3
