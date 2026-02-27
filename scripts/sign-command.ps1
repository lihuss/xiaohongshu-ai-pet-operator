param(
  [Parameter(Mandatory = $true)][string]$ActorUserId,
  [Parameter(Mandatory = $true)][string]$Command,
  [Parameter(Mandatory = $true)][string]$Secret,
  [string]$ArgsJson = "{}"
)

$timestamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$nonce = [guid]::NewGuid().ToString("N")

$argsObj = $ArgsJson | ConvertFrom-Json
$argsCanonical = $argsObj | ConvertTo-Json -Compress
$base = "$ActorUserId`n$Command`n$argsCanonical`n$timestamp`n$nonce"

$hmac = New-Object System.Security.Cryptography.HMACSHA256
$hmac.Key = [Text.Encoding]::UTF8.GetBytes($Secret)
$hash = $hmac.ComputeHash([Text.Encoding]::UTF8.GetBytes($base))
$sig = -join ($hash | ForEach-Object { $_.ToString("x2") })

$payload = @{
  actor_user_id = $ActorUserId
  command = $Command
  args = ($ArgsJson | ConvertFrom-Json)
  timestamp = $timestamp
  nonce = $nonce
  signature = $sig
}

$payload | ConvertTo-Json -Depth 10

