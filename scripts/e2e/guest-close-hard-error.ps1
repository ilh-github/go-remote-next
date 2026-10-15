# Dismiss the "app.exe - System Error" hard-error dialog that csrss shows when
# a process fails to start (here: file-ops.exe without vcruntime140.dll).
param([string]$TitlePart = 'file-ops')

Add-Type @'
using System;
using System.Text;
using System.Runtime.InteropServices;
public class QdClose {
    public delegate bool EnumProc(IntPtr h, IntPtr l);
    [DllImport("user32.dll")] public static extern bool EnumWindows(EnumProc cb, IntPtr l);
    [DllImport("user32.dll", CharSet = CharSet.Unicode)] public static extern int GetWindowText(IntPtr h, StringBuilder s, int n);
    [DllImport("user32.dll")] public static extern IntPtr SendMessageTimeout(IntPtr h, uint msg, IntPtr w, IntPtr l, uint flags, uint timeout, out IntPtr result);
}
'@

$hit = New-Object System.Collections.ArrayList
$callback = [QdClose+EnumProc]{
    param($h, $l)
    $sb = New-Object System.Text.StringBuilder 512
    [void][QdClose]::GetWindowText($h, $sb, 512)
    if ($sb.Length -gt 0 -and $sb.ToString() -like "*$TitlePart*") {
        [void]$hit.Add(("{0}|{1}" -f $h, $sb.ToString()))
        # WM_COMMAND with IDOK, then WM_CLOSE as a fallback.
        $r = [IntPtr]::Zero
        [void][QdClose]::SendMessageTimeout($h, 0x0111, [IntPtr]1, [IntPtr]::Zero, 2, 2000, [ref]$r)
        [void][QdClose]::SendMessageTimeout($h, 0x0010, [IntPtr]::Zero, [IntPtr]::Zero, 2, 2000, [ref]$r)
    }
    return $true
}
[void][QdClose]::EnumWindows($callback, [IntPtr]::Zero)
if ($hit.Count -eq 0) { Write-Output 'no matching window' } else { $hit | ForEach-Object { Write-Output $_ } }
