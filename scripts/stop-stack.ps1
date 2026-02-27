$ErrorActionPreference = "SilentlyContinue"

$ports = @(8081, 18060)
foreach ($port in $ports) {
  $conn = Get-NetTCPConnection -LocalPort $port -State Listen
  if ($conn) {
    Stop-Process -Id $conn.OwningProcess -Force
    Write-Output "stopped listener on :$port (pid=$($conn.OwningProcess))"
  }
}

