<#
Guest-side helper for the Parallels Desktop Windows verification baseline.

Runs inside the Windows guest and prints exactly one machine readable line:
    PDJSON_B64:<base64(utf8 json)>
Everything else printed by the script is diagnostic noise for the host log.
#>
param(
  [Parameter(Mandatory = $true)]
  [ValidateSet('probe', 'launch', 'keyboard', 'file', 'display', 'lock', 'netstat')]
  [string]$Action,
  [string]$ExePath = '',
  [string]$AppArgs = '',
  [string]$Dir = '',
  [string]$WaitFile = '',
  [string]$ExpectSha256 = '',
  [string]$CreateFile = '',
  [string]$CreateContent = '',
  [string]$HostAddr = '10.211.55.2',
  [string]$ProcessName = '',
  [int]$Port = 0,
  [int]$TimeoutSec = 60,
  [int]$WaitSec = 0,
  [switch]$SelfTest,
  [switch]$Force,
  [switch]$KeepOpen
)

$ErrorActionPreference = 'Stop'

function Write-Result {
  param($Object)
  $json = $Object | ConvertTo-Json -Depth 8 -Compress
  $bytes = [System.Text.Encoding]::UTF8.GetBytes($json)
  Write-Output ('PDJSON_B64:' + [Convert]::ToBase64String($bytes))
}

function Add-Win32Types {
  if (-not ('PD.Win32' -as [type])) {
    Add-Type -Namespace PD -Name Win32 -MemberDefinition @'
[DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
[DllImport("user32.dll", CharSet = CharSet.Unicode)] public static extern int GetWindowText(IntPtr hWnd, System.Text.StringBuilder lpString, int nMaxCount);
[DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr hWnd, out uint lpdwProcessId);
'@
  }
}

function Get-ForegroundInfo {
  Add-Win32Types
  $hwnd = [PD.Win32]::GetForegroundWindow()
  if ($hwnd -eq [IntPtr]::Zero) {
    return @{ hwnd = 0; pid = 0; process = ''; title = '' }
  }
  $sb = New-Object System.Text.StringBuilder 512
  [void][PD.Win32]::GetWindowText($hwnd, $sb, $sb.Capacity)
  $procId = 0
  [void][PD.Win32]::GetWindowThreadProcessId($hwnd, [ref]$procId)
  $name = ''
  try { $name = (Get-Process -Id $procId -ErrorAction Stop).ProcessName } catch { $name = '' }
  return @{
    hwnd    = [int64]$hwnd
    pid     = [int]$procId
    process = $name
    title   = $sb.ToString()
  }
}

function Get-DisplayList {
  $adapters = @()
  foreach ($a in (Get-CimInstance Win32_VideoController)) {
    $adapters += @{
      name   = $a.Name
      width  = [int]$a.CurrentHorizontalResolution
      height = [int]$a.CurrentVerticalResolution
    }
  }
  # The comma keeps single-element collections as arrays through ConvertTo-Json.
  return ,$adapters
}

function Get-LockState {
  $logon = @(Get-Process -Name LogonUI -ErrorAction SilentlyContinue)
  return @{
    locked     = ($logon.Count -gt 0)
    logonUiPid = if ($logon.Count -gt 0) { [int]$logon[0].Id } else { 0 }
    method     = 'LogonUI.exe presence'
  }
}

function Get-ActiveSession {
  $userName = $null
  try { $userName = (Get-CimInstance Win32_ComputerSystem).UserName } catch { }
  $id = ''
  $name = ''
  $state = ''
  try {
    $line = (quser 2>$null | Select-Object -Skip 1 | Select-Object -First 1)
    if ($line) {
      $clean = $line.TrimStart('>', ' ')
      if ($clean -match '^(?<user>\S+)\s+(?<session>\S+)\s+(?<id>\d+)\s+(?<state>\S+)') {
        $name = $Matches['session']
        $id = $Matches['id']
        $state = $Matches['state']
      }
    }
  } catch { }
  return @{ id = $id; name = $name; state = $state; userName = $userName }
}

function Get-HostReachability {
  param([string]$Target, [int]$TargetPort)
  $result = @{ ok = $false; detail = '' }
  $ping = Test-Connection -ComputerName $Target -Count 1 -Quiet -ErrorAction SilentlyContinue
  $result.ok = [bool]$ping
  $result.detail = "ping=$ping"
  if ($TargetPort -gt 0) {
    $tcp = $null
    try {
      $client = New-Object System.Net.Sockets.TcpClient
      $iar = $client.BeginConnect($Target, $TargetPort, $null, $null)
      $ok = $iar.AsyncWaitHandle.WaitOne(2000, $false)
      if ($ok) { $client.EndConnect($iar) }
      $tcp = $ok
      $client.Close()
    } catch { $tcp = $false }
    $result.ok = $result.ok -and [bool]$tcp
    $result.detail = "$($result.detail) tcp/$TargetPort=$tcp"
  }
  return $result
}

function Get-UiaWindow {
  param([int]$ProcessId)
  Add-Type -AssemblyName UIAutomationClient, UIAutomationTypes
  $root = [System.Windows.Automation.AutomationElement]::RootElement
  $cond = New-Object System.Windows.Automation.PropertyCondition(
    [System.Windows.Automation.AutomationElement]::ProcessIdProperty, $ProcessId)
  return $root.FindFirst([System.Windows.Automation.TreeScope]::Children, $cond)
}

function Get-DocumentText {
  param($Window)
  if (-not $Window) { return @{ ok = $false; text = $null; via = ''; error = 'no window element' } }
  foreach ($type in @([System.Windows.Automation.ControlType]::Document, [System.Windows.Automation.ControlType]::Edit)) {
    $cond = New-Object System.Windows.Automation.PropertyCondition(
      [System.Windows.Automation.AutomationElement]::ControlTypeProperty, $type)
    $element = $Window.FindFirst([System.Windows.Automation.TreeScope]::Descendants, $cond)
    if (-not $element) { continue }
    try {
      $value = $element.GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern)
      return @{ ok = $true; text = $value.Current.Value; via = 'ValuePattern'; controlType = $type.ProgrammaticName }
    } catch { }
    try {
      $text = $element.GetCurrentPattern([System.Windows.Automation.TextPattern]::Pattern)
      return @{ ok = $true; text = $text.DocumentRange.GetText(-1); via = 'TextPattern'; controlType = $type.ProgrammaticName }
    } catch { }
  }
  return @{ ok = $false; text = $null; via = ''; error = 'no ValuePattern/TextPattern document found' }
}

function Invoke-Probe {
  $os = Get-CimInstance Win32_OperatingSystem
  $cs = Get-CimInstance Win32_ComputerSystem
  $shared = @()
  try { $shared = @(Get-CimInstance Win32_NetworkConnection -ErrorAction Stop | ForEach-Object { "$($_.LocalName) -> $($_.RemoteName) [$($_.ConnectionState)]" }) } catch { }
  if ($shared.Count -eq 0) {
    $shared = @((net use 2>$null) | Where-Object { $_ -match '\\\\|:.*\\\\' })
  }
  $installed = @{}
  foreach ($path in @('C:\Program Files\QuickDesk', 'C:\Program Files (x86)\QuickDesk', 'C:\QuickDeskTest\artifacts', 'C:\QuickDeskTest\bin')) {
    $installed[$path] = (Test-Path $path)
  }
  $lock = Get-LockState
  $session = Get-ActiveSession
  return @{
    action          = 'probe'
    ok              = $true
    time            = (Get-Date).ToString('s')
    hostname        = $env:COMPUTERNAME
    user            = "$env:USERDOMAIN\$env:USERNAME"
    os              = @{
      caption = $os.Caption
      version = $os.Version
      build   = "$($os.BuildNumber)"
      arch    = $os.OSArchitecture
    }
    computer        = @{
      model      = $cs.Model
      systemType = $cs.SystemType
      ramGB      = [math]::Round($cs.TotalPhysicalMemory / 1GB, 1)
    }
    session         = $session
    locked          = $lock.locked
    lockMethod      = $lock.method
    lockPid         = $lock.logonUiPid
    displays        = Get-DisplayList
    foreground      = Get-ForegroundInfo
    powershell      = @{ version = $PSVersionTable.PSVersion.ToString() }
    shared          = $shared
    installed       = $installed
    hostReachability = Get-HostReachability -Target $HostAddr -TargetPort $Port
  }
}

function Invoke-Launch {
  if (-not (Test-Path $ExePath)) {
    return @{ action = 'launch'; ok = $false; error = "not found: $ExePath" }
  }
  $startParams = @{ FilePath = $ExePath; PassThru = $true }
  if ($AppArgs) { $startParams.ArgumentList = $AppArgs }
  $started = Start-Process @startParams
  Start-Sleep -Seconds ([math]::Max(2, [math]::Min($WaitSec, 10)))
  $alive = $true
  try { $proc = Get-Process -Id $started.Id -ErrorAction Stop } catch {
    $alive = $false
    $proc = $null
  }
  $conns = @()
  if ($HostAddr) {
    try {
      $conns = @(Get-NetTCPConnection -ErrorAction Stop |
        Where-Object { $_.RemoteAddress -eq $HostAddr -and $_.State -eq 'Established' } |
        Select-Object -First 20 |
        ForEach-Object { "$($_.LocalAddress):$($_.LocalPort) -> $($_.RemoteAddress):$($_.RemotePort)" })
    } catch { }
  }
  return @{
    action      = 'launch'
    ok          = $alive
    exe         = $ExePath
    args        = $AppArgs
    pid         = $started.Id
    alive       = $alive
    processName = if ($proc) { $proc.ProcessName } else { '' }
    foreground  = Get-ForegroundInfo
    connections = $conns
  }
}

function Get-NotepadWindow {
  # Windows 11 Notepad may be a launcher + app pair, so match by process name.
  $deadline = (Get-Date).AddSeconds(20)
  while ((Get-Date) -lt $deadline) {
    foreach ($p in @(Get-Process -Name Notepad -ErrorAction SilentlyContinue | Where-Object { $_.MainWindowHandle -ne 0 })) {
      $w = Get-UiaWindow -ProcessId $p.Id
      if ($w) { return @{ window = $w; process = $p } }
    }
    Start-Sleep -Milliseconds 500
  }
  return $null
}

function Close-NotepadSince {
  param([datetime]$Since)
  Get-Process -Name Notepad -ErrorAction SilentlyContinue |
    Where-Object { $_.StartTime -ge $Since } |
    ForEach-Object { Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue }
}

function Get-TextViaClipboard {
  # WinUI Notepad does not always expose Value/TextPattern, so fall back to a
  # select-all + copy round trip and restore the previous clipboard content.
  param($Window)
  Add-Type -AssemblyName System.Windows.Forms
  $savedClip = $null
  try { $savedClip = Get-Clipboard -Raw -ErrorAction Stop } catch { }
  try { $Window.SetFocus() } catch { }
  Start-Sleep -Milliseconds 300
  [System.Windows.Forms.SendKeys]::SendWait('^a')
  Start-Sleep -Milliseconds 200
  [System.Windows.Forms.SendKeys]::SendWait('^c')
  Start-Sleep -Milliseconds 500
  $text = $null
  try { $text = Get-Clipboard -Raw -ErrorAction Stop } catch { }
  $restored = $false
  if ($null -ne $savedClip) {
    try { Set-Clipboard -Value $savedClip; $restored = $true } catch { }
  }
  return @{ text = $text; clipboardRestored = $restored }
}

function Invoke-Keyboard {
  $before = Get-ForegroundInfo
  $startedAt = Get-Date
  try {
    Start-Process -FilePath 'notepad.exe' | Out-Null
  } catch {
    return @{ action = 'keyboard'; ok = $false; error = "failed to start notepad.exe: $($_.Exception.Message)" }
  }
  $found = Get-NotepadWindow
  if (-not $found) {
    if (-not $KeepOpen) { Close-NotepadSince -Since $startedAt }
    return @{ action = 'keyboard'; ok = $false; error = 'notepad window not found via UI Automation'; foregroundBefore = $before }
  }
  $window = $found.window
  $notepadPid = $found.process.Id
  $initial = Get-DocumentText -Window $window
  $initialTitle = "$((Get-Process -Id $notepadPid -ErrorAction SilentlyContinue).MainWindowTitle)"
  if ($SelfTest) {
    Add-Type -AssemblyName System.Windows.Forms
    $element = $window.FindFirst([System.Windows.Automation.TreeScope]::Descendants,
      (New-Object System.Windows.Automation.PropertyCondition(
        [System.Windows.Automation.AutomationElement]::ControlTypeProperty,
        [System.Windows.Automation.ControlType]::Document)))
    if ($element) { try { $element.SetFocus() } catch { } }
    Start-Sleep -Milliseconds 500
    [System.Windows.Forms.SendKeys]::SendWait('pd-selftest')
  }

  # Fast, non-invasive signal: Notepad marks the title dirty once input lands.
  $titleChanged = $false
  $deadline = (Get-Date).AddSeconds([math]::Max(5, $TimeoutSec))
  while ((Get-Date) -lt $deadline) {
    Start-Sleep -Milliseconds 750
    $proc = Get-Process -Id $notepadPid -ErrorAction SilentlyContinue
    if ($proc -and $proc.MainWindowTitle -and $proc.MainWindowTitle -ne $initialTitle) {
      $titleChanged = $true
      break
    }
  }
  $titleAfter = "$((Get-Process -Id $notepadPid -ErrorAction SilentlyContinue).MainWindowTitle)"

  # Authoritative read of what landed in the focused control.
  $result_clipboardRestored = $false
  $final = Get-DocumentText -Window $window
  $capturedVia = $final.via
  $captureError = $final.error
  $text = $final.text
  if (-not $final.ok) {
    $clip = Get-TextViaClipboard -Window $window
    if ($clip.text) {
      $text = $clip.text
      $capturedVia = 'ClipboardCtrlAC'
      $captureError = ''
      $result_clipboardRestored = $clip.clipboardRestored
    }
  }
  $textArrived = $false
  if ($text) { $textArrived = ($text.Trim().Length -gt 0) -and ($text -ne $initial.text) }
  $ok = $titleChanged -or $textArrived
  $focusNow = Get-ForegroundInfo
  if (-not $KeepOpen) { Close-NotepadSince -Since $startedAt }
  return @{
    action            = 'keyboard'
    ok                = $ok
    selfTest          = [bool]$SelfTest
    notepadPid        = $notepadPid
    titleChanged      = $titleChanged
    titleBefore       = $initialTitle
    titleAfter        = $titleAfter
    capturedVia       = $capturedVia
    capturedType      = $final.controlType
    captureError      = $captureError
    clipboardRestored = [bool]$result_clipboardRestored
    text              = $text
    textLength        = if ($text) { $text.Length } else { 0 }
    foregroundBefore  = $before
    foregroundAfter   = $focusNow
    focusKeptOnApp    = ($focusNow.process -match 'notepad')
  }
}

function Invoke-File {
  $result = @{ action = 'file'; ok = $false }
  if ($Dir) {
    if (-not (Test-Path $Dir)) { New-Item -ItemType Directory -Path $Dir -Force | Out-Null }
    $result.dir = $Dir
  }
  if ($CreateFile -and $Dir) {
    $target = Join-Path $Dir $CreateFile
    $bytes = [System.Text.Encoding]::ASCII.GetBytes($CreateContent)
    [System.IO.File]::WriteAllBytes($target, $bytes)
    $hash = (Get-FileHash -Algorithm SHA256 -Path $target).Hash.ToLower()
    $result.created = @{ path = $target; bytes = $bytes.Length; sha256 = $hash }
  }
  if ($WaitFile -and $Dir) {
    $target = Join-Path $Dir $WaitFile
    $deadline = (Get-Date).AddSeconds([math]::Max(5, $TimeoutSec))
    while ((Get-Date) -lt $deadline -and -not (Test-Path $target)) {
      Start-Sleep -Milliseconds 500
    }
    if (-not (Test-Path $target)) {
      $result.ok = $false
      $result.error = "file did not arrive: $target"
      return $result
    }
    $item = Get-Item $target
    $hash = (Get-FileHash -Algorithm SHA256 -Path $target).Hash.ToLower()
    $matched = $true
    if ($ExpectSha256) { $matched = ($hash -eq $ExpectSha256.ToLower()) }
    $result.waitFile = @{ path = $target; size = $item.Length; sha256 = $hash; hashMatched = $matched }
    $result.ok = $matched
  } else {
    $result.ok = $true
  }
  if ($Dir -and (Test-Path $Dir)) {
    $result.files = @(Get-ChildItem $Dir -File | Select-Object -First 20 | ForEach-Object { "$($_.Name) ($($_.Length)B)" })
  }
  return $result
}

function Invoke-Display {
  $before = Get-DisplayList
  if ($WaitSec -gt 0) { Start-Sleep -Seconds $WaitSec }
  $after = Get-DisplayList
  $changed = ($before | ConvertTo-Json -Compress) -ne ($after | ConvertTo-Json -Compress)
  return @{ action = 'display'; ok = $true; changed = $changed; before = $before; after = $after }
}

function Invoke-Lock {
  $before = Get-LockState
  $result = @{ action = 'lock'; before = $before.locked; beforePid = $before.logonUiPid; method = $before.method }
  $desktop = Get-ItemProperty -Path 'HKCU:\Control Panel\Desktop' -ErrorAction SilentlyContinue
  if ($desktop) {
    $result.screenSaver = @{
      active  = "$($desktop.ScreenSaveActive)"
      timeout = "$($desktop.ScreenSaveTimeOut)"
      secure  = "$($desktop.ScreenSaverIsSecure)"
    }
  }
  if ($Force) {
    Start-Process -FilePath 'rundll32.exe' -ArgumentList 'user32.dll,LockWorkStation' | Out-Null
  }
  if ($WaitSec -gt 0 -or $Force) {
    $deadline = (Get-Date).AddSeconds([math]::Max($WaitSec, 10))
    $observed = $false
    while ((Get-Date) -lt $deadline) {
      Start-Sleep -Milliseconds 750
      if ((Get-LockState).locked) { $observed = $true; break }
    }
    $result.lockObserved = $observed
  }
  $after = Get-LockState
  $result.after = $after.locked
  $result.afterPid = $after.logonUiPid
  $result.ok = $true
  return $result
}

function Invoke-Netstat {
  $rows = @()
  try {
    $rows = @(Get-NetTCPConnection -ErrorAction Stop | Where-Object {
        $_.State -eq 'Established' -and (-not $HostAddr -or $_.RemoteAddress -eq $HostAddr) -and ($Port -eq 0 -or $_.RemotePort -eq $Port)
      } | Select-Object -First 40 | ForEach-Object {
        "local=$($_.LocalAddress):$($_.LocalPort) remote=$($_.RemoteAddress):$($_.RemotePort) pid=$($_.OwningProcess) state=$($_.State)"
      })
  } catch {
    $rows = @((netstat -ano) | Where-Object { $_ -match 'ESTABLISHED' -and ($Port -eq 0 -or $_ -match ":$Port\s") } | Select-Object -First 40)
  }
  return @{ action = 'netstat'; ok = $true; hostAddr = $HostAddr; port = $Port; count = $rows.Count; connections = $rows }
}

try {
  switch ($Action) {
    'probe' { Write-Result (Invoke-Probe) }
    'launch' { Write-Result (Invoke-Launch) }
    'keyboard' { Write-Result (Invoke-Keyboard) }
    'file' { Write-Result (Invoke-File) }
    'display' { Write-Result (Invoke-Display) }
    'lock' { Write-Result (Invoke-Lock) }
    'netstat' { Write-Result (Invoke-Netstat) }
  }
} catch {
  Write-Result @{ action = $Action; ok = $false; error = $_.Exception.Message; stack = $_.ScriptStackTrace }
  exit 1
}
