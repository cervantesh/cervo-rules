param(
    [Parameter(Mandatory = $true, Position = 0)]
    [string]$Version,

    [Parameter(Mandatory = $false, Position = 1)]
    [string]$DistDir = "dist-release-check"
)

$ErrorActionPreference = "Stop"

$scriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Resolve-Path (Join-Path $scriptRoot "..\..")
$releaseScript = Join-Path $repoRoot "scripts/release/check.sh"

function Resolve-Bash {
    $candidates = @(
        "C:\Program Files\Git\bin\bash.exe",
        "C:\Program Files\Git\usr\bin\bash.exe"
    )

    foreach ($candidate in $candidates) {
        if (Test-Path -LiteralPath $candidate) {
            return $candidate
        }
    }

    $command = Get-Command bash -ErrorAction SilentlyContinue
    if ($null -ne $command) {
        if ($command.Source -like "*\Windows\System32\bash.exe") {
            throw "Found Windows WSL launcher at '$($command.Source)', but no Git Bash executable. Install Git for Windows or run scripts/release/check.sh from a real bash shell."
        }
        return $command.Source
    }

    throw "No bash executable found. Install Git for Windows or run scripts/release/check.sh from a shell that provides bash."
}

$bash = Resolve-Bash

Push-Location $repoRoot
try {
    & $bash $releaseScript $Version $DistDir
    exit $LASTEXITCODE
}
finally {
    Pop-Location
}
