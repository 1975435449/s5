param(
    [string]$HostUrl = "http://127.0.0.1:18080"
)

$ErrorActionPreference = "Stop"

$machinePath = [System.Environment]::GetEnvironmentVariable("PATH", "Machine")
$userPath = [System.Environment]::GetEnvironmentVariable("PATH", "User")
$env:PATH = "$machinePath;$userPath"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "Go is not available in PATH. Close and reopen PowerShell, or install Go first." -ForegroundColor Red
    Write-Host "Chocolatey install command: choco install golang -y"
    exit 1
}

$conf = Join-Path $PSScriptRoot "conf\nps.conf"
if (-not (Test-Path -LiteralPath $conf)) {
    Write-Host "Missing config file: $conf" -ForegroundColor Red
    exit 1
}

foreach ($port in 18080, 8024) {
    $conn = Get-NetTCPConnection -LocalPort $port -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($conn) {
        $proc = Get-Process -Id $conn.OwningProcess -ErrorAction SilentlyContinue
        Write-Host "Port $port is already used by PID $($conn.OwningProcess) $($proc.ProcessName)" -ForegroundColor Yellow
        Write-Host "Stop that process first, or change the port in conf\nps.conf."
        exit 1
    }
}

$exe = Join-Path $PSScriptRoot "nps-local.exe"

Write-Host "Go: $(go version)"
Write-Host "Building local nps binary..."
Set-Location -LiteralPath $PSScriptRoot
go build -o $exe ./cmd/nps/nps.go

Write-Host "Starting nps with conf_path=$PSScriptRoot"
Write-Host "Web panel: $HostUrl"
Write-Host "Default login: admin / admin"
Write-Host "Keep this PowerShell window open. Press Ctrl+C to stop."

& $exe
