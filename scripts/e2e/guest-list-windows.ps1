param([int]$OwnerPid = 0)

Add-Type @'
using System;
using System.Text;
using System.Runtime.InteropServices;
public class QdWin {
    public delegate bool EnumProc(IntPtr h, IntPtr l);
    [DllImport("user32.dll")] public static extern bool EnumWindows(EnumProc cb, IntPtr l);
    [DllImport("user32.dll", CharSet = CharSet.Unicode)] public static extern int GetWindowText(IntPtr h, StringBuilder s, int n);
    [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr h);
    [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
    [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out RECT r);
    public struct RECT { public int Left, Top, Right, Bottom; }
}
'@

$results = New-Object System.Collections.ArrayList
$callback = [QdWin+EnumProc]{
    param($h, $l)
    $sb = New-Object System.Text.StringBuilder 512
    [void][QdWin]::GetWindowText($h, $sb, 512)
    $owner = 0
    [void][QdWin]::GetWindowThreadProcessId($h, [ref]$owner)
    if ($sb.Length -gt 0) {
        if ($OwnerPid -eq 0 -or $owner -eq $OwnerPid) {
            $rect = New-Object QdWin+RECT
            [void][QdWin]::GetWindowRect($h, [ref]$rect)
            [void]$results.Add(("pid={0} hwnd={1} visible={2} rect={3},{4},{5},{6} title={7}" -f `
                $owner, $h, [QdWin]::IsWindowVisible($h), $rect.Left, $rect.Top, $rect.Right, $rect.Bottom, $sb.ToString()))
        }
    }
    return $true
}
[void][QdWin]::EnumWindows($callback, [IntPtr]::Zero)
$results | Sort-Object | ForEach-Object { Write-Output $_ }
