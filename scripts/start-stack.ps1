param(
  [string]$ConfigPath = ".\\config\\user.config.json"
)

$ErrorActionPreference = "Stop"

if (!(Test-Path $ConfigPath)) {
  throw "Missing config file: $ConfigPath"
}

$cfg = Get-Content $ConfigPath -Raw | ConvertFrom-Json
$root = Resolve-Path (Join-Path $PSScriptRoot "..")
$logDir = Join-Path $root "logs"
if (!(Test-Path $logDir)) {
  New-Item -ItemType Directory -Path $logDir | Out-Null
}

$go = "C:\Program Files\Go\bin\go.exe"
if (!(Test-Path $go)) {
  throw "Go not found: $go"
}

$mcpDir = Join-Path $root "third_party\xiaohongshu-mcp"
$opDir = $root

$mcpOut = Join-Path $logDir "mcp.out.log"
$mcpErr = Join-Path $logDir "mcp.err.log"
$opOut = Join-Path $logDir "operator.out.log"
$opErr = Join-Path $logDir "operator.err.log"

$mcpScript = @"
`$env:ROD_LEAKLESS='false'
& '$go' run .
"@
$mcpRun = Join-Path $mcpDir "run-local.ps1"
$mcpScript | Set-Content -Encoding UTF8 $mcpRun

$opScript = @"
& '$go' run .\cmd\server\main.go
"@
$opRun = Join-Path $opDir "run-local.ps1"
$opScript | Set-Content -Encoding UTF8 $opRun

Start-Process -FilePath powershell `
  -ArgumentList "-NoProfile","-ExecutionPolicy","Bypass","-File",$mcpRun `
  -WorkingDirectory $mcpDir `
  -RedirectStandardOutput $mcpOut `
  -RedirectStandardError $mcpErr | Out-Null

Start-Sleep -Seconds 3

Start-Process -FilePath powershell `
  -ArgumentList "-NoProfile","-ExecutionPolicy","Bypass","-File",$opRun `
  -WorkingDirectory $opDir `
  -RedirectStandardOutput $opOut `
  -RedirectStandardError $opErr | Out-Null

Start-Sleep -Seconds 2

Write-Output "stack started"
Write-Output "operator: http://127.0.0.1$($cfg.operator.listen_addr)/healthz"
Write-Output "mcp: $($cfg.mcp.base_url)/health"

