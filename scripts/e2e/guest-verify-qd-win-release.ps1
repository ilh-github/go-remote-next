param(
    [string]$PackageDir = 'C:\qd-build-src\go-remote-next\publish\Release',
    [string]$VerifierPath = (Join-Path $PSScriptRoot '..\verify_qd_win_release.ps1'),
    [string]$DumpbinPath = ''
)

$ErrorActionPreference = 'Stop'

function Invoke-PublishVerifier {
    param([string]$Path)

    $arguments = @('-File', $Path, '-PackageDir', $PackageDir)
    if ($DumpbinPath) {
        $arguments += @('-DumpbinPath', $DumpbinPath)
    }
    $previousErrorAction = $ErrorActionPreference
    try {
        $ErrorActionPreference = 'Continue'
        $output = & powershell.exe -NoProfile -ExecutionPolicy Bypass @arguments 2>&1
        $exitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $previousErrorAction
    }
    return @{
        ExitCode = $exitCode
        Output = ($output -join [Environment]::NewLine)
    }
}

if (-not (Test-Path -LiteralPath $VerifierPath -PathType Leaf)) {
    throw "publish verifier is missing: $VerifierPath"
}
if (-not (Test-Path -LiteralPath $PackageDir -PathType Container)) {
    throw "publish directory is missing: $PackageDir"
}

$positive = Invoke-PublishVerifier -Path $VerifierPath
if ($positive.ExitCode -ne 0) {
    throw "valid publish package was rejected:`n$($positive.Output)"
}

function Assert-MissingFileRejected {
    param(
        [string]$RelativePath,
        [string]$ExpectedPattern
    )

    $file = Join-Path $PackageDir $RelativePath
    $backup = "$file.qd-verify-backup"
    if (-not (Test-Path -LiteralPath $file -PathType Leaf)) {
        throw "test fixture is missing: $file"
    }
    if (Test-Path -LiteralPath $backup) {
        Remove-Item -LiteralPath $backup -Force
    }

    Rename-Item -LiteralPath $file -NewName (Split-Path -Leaf $backup)
    try {
        $negative = Invoke-PublishVerifier -Path $VerifierPath
    } finally {
        Rename-Item -LiteralPath $backup -NewName (Split-Path -Leaf $file)
    }

    if ($negative.ExitCode -eq 0) {
        throw "publish verifier accepted a package with $RelativePath removed"
    }
    if ($negative.Output -notmatch $ExpectedPattern) {
        throw "missing-file failure did not name ${RelativePath}:`n$($negative.Output)"
    }
}

Assert-MissingFileRejected `
    -RelativePath 'vcruntime140_1.dll' `
    -ExpectedPattern 'vcruntime140_1'
Assert-MissingFileRejected `
    -RelativePath 'platforms\qwindows.dll' `
    -ExpectedPattern 'qwindows'
Assert-MissingFileRejected `
    -RelativePath 'Qt6Network.dll' `
    -ExpectedPattern 'Qt6Network'

Write-Host 'PASS: valid package accepted; missing runtime, Qt plugin and Qt dependency rejected'
