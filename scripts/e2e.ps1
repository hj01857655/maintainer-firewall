param(
  [string]$AdminUsername = "admin",
  [string]$AdminPassword = "CHANGE_ME_ADMIN_PASSWORD",
  [string]$JWTSecret = "CHANGE_ME_JWT_SECRET",
  [string]$GitHubWebhookSecret = "CHANGE_ME_WEBHOOK_SECRET",
  [string]$GitHubToken = "",
  [string]$GitHubTestRepo = "",
  [int]$GitHubTestIssueNumber = 0,
  [string]$DatabaseURL = "postgres://<user>:<password>@localhost:5432/maintainer_firewall?sslmode=disable",
  [int]$ApiPort = 8080,
  [int]$WebPort = 5173,
  [switch]$StartWeb,
  [switch]$KeepApiRunning,
  [switch]$KeepWebRunning
)

$ErrorActionPreference = "Stop"

function Assert-True {
  param([bool]$Condition, [string]$Message)
  if (-not $Condition) {
    throw "[ASSERT FAILED] $Message"
  }
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$apiDir = Join-Path $repoRoot "apps\api-go"
$webDir = Join-Path $repoRoot "apps\web-react"

Write-Host "[E2E 1/8] Set environment variables..." -ForegroundColor Cyan

$env:ADMIN_USERNAME = $AdminUsername
$env:ADMIN_PASSWORD = $AdminPassword
$env:JWT_SECRET = $JWTSecret
$env:GITHUB_WEBHOOK_SECRET = $GitHubWebhookSecret
$env:GITHUB_TOKEN = $GitHubToken
$env:DATABASE_URL = $DatabaseURL
$env:PORT = "$ApiPort"

Write-Host "[E2E 2/8] Start API..." -ForegroundColor Cyan

$apiProc = Start-Process -FilePath "powershell" -ArgumentList "-NoProfile -Command go run .\cmd\server\main.go" -WorkingDirectory $apiDir -PassThru

$webProc = $null
if ($StartWeb) {
  Write-Host "[E2E 2.5/8] Start Web..." -ForegroundColor Cyan
  $webProc = Start-Process -FilePath "powershell" -ArgumentList "-NoProfile -Command npm exec vite -- --port=$WebPort" -WorkingDirectory $webDir -PassThru
}


if ($AdminPassword -like 'CHANGE_ME*' -or $JWTSecret -like 'CHANGE_ME*' -or $GitHubWebhookSecret -like 'CHANGE_ME*') {
  throw 'Please provide real secrets via parameters: -AdminPassword -JWTSecret -GitHubWebhookSecret'
}
if ([string]::IsNullOrWhiteSpace($AdminUsername)) {
  throw 'Please provide -AdminUsername'
}
if ($DatabaseURL -like 'postgres://<user>:<password>*') {
  throw 'Please provide real PostgreSQL URL via -DatabaseURL'
}
if ($DatabaseURL -like 'mysql://<*') {
  throw 'Please provide real MySQL URL via -DatabaseURL'
}
if ([string]::IsNullOrWhiteSpace($GitHubTestRepo)) {
  throw 'Please provide real GitHub repo via -GitHubTestRepo (format: owner/repo)'
}
if ($GitHubTestIssueNumber -le 0) {
  throw 'Please provide real issue number via -GitHubTestIssueNumber (>0)'
}
$cleanupApi = -not $KeepApiRunning
$cleanupWeb = $StartWeb -and (-not $KeepWebRunning)
$apiHealthUrl = "http://localhost:$ApiPort/health"
$webHealthUrl = "http://localhost:$WebPort/login"
try {
Write-Host "[E2E 3/8] Wait API health..." -ForegroundColor Cyan

  for ($i = 0; $i -lt 60; $i++) {
    try {
      $health = Invoke-RestMethod -Method Get -Uri $apiHealthUrl -TimeoutSec 2
      Assert-True ($health.status -eq "ok") "health status should be ok"
      break
    } catch {
      Start-Sleep -Milliseconds 500
    }
    if ($i -eq 59) { throw "API did not become healthy in time" }
  }

if ($StartWeb) {
    Write-Host "[E2E 3.5/8] Wait Web ready (port=$WebPort)..." -ForegroundColor Cyan
    for ($i = 0; $i -lt 60; $i++) {
      try {
        $resp = Invoke-WebRequest -Method Get -Uri $webHealthUrl -TimeoutSec 2 -UseBasicParsing
        if ($resp.StatusCode -ge 200 -and $resp.StatusCode -lt 500) { break }
      } catch {
        Start-Sleep -Milliseconds 500
      }
      if ($i -eq 59) { throw "Web did not become ready in time" }
    }
    Write-Host "Web ready: http://localhost:$WebPort/login" -ForegroundColor Green
    Write-Host "Default test account: $AdminUsername / (password from -AdminPassword)" -ForegroundColor DarkGray
  }

Write-Host "[E2E 4/8] Login... (username=$AdminUsername)" -ForegroundColor Cyan
  $loginBody = @{ username = $AdminUsername; password = $AdminPassword } | ConvertTo-Json
  $login = Invoke-RestMethod -Method Post -Uri "http://localhost:$ApiPort/auth/login" -ContentType "application/json" -Body $loginBody

  Assert-True ($login.ok -eq $true) "login should return ok=true"
  Assert-True (-not [string]::IsNullOrWhiteSpace($login.token)) "login token should not be empty"
  $authHeaders = @{ Authorization = "Bearer $($login.token)" }

  Write-Host "[E2E 5/8] Create rule..." -ForegroundColor Cyan
  $ruleBody = @{
    event_type = "issues"
    keyword = "urgent"
    suggestion_type = "label"
    suggestion_value = "priority-high"
    reason = "e2e urgent rule"
    is_active = $true
  } | ConvertTo-Json
  $ruleResp = Invoke-RestMethod -Method Post -Uri "http://localhost:$ApiPort/rules" -Headers $authHeaders -ContentType "application/json" -Body $ruleBody


  Write-Host "[E2E 6/8] Send webhook..." -ForegroundColor Cyan
  $deliveryId = "e2e-" + [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
  $payloadObj = @{
    action = "opened"
    repository = @{ full_name = $GitHubTestRepo }
    sender = @{ login = "alice" }
    issue = @{ number = $GitHubTestIssueNumber; title = "urgent issue from e2e" }
  }
  $payload = $payloadObj | ConvertTo-Json -Compress
  $hmac = New-Object System.Security.Cryptography.HMACSHA256
  $hmac.Key = [Text.Encoding]::UTF8.GetBytes($GitHubWebhookSecret)
  $hash = $hmac.ComputeHash([Text.Encoding]::UTF8.GetBytes($payload))
  $signature = "sha256=" + ([BitConverter]::ToString($hash).Replace("-", "").ToLower())

  $webhookHeaders = @{
    "X-Hub-Signature-256" = $signature
    "X-GitHub-Event" = "issues"
    "X-GitHub-Delivery" = $deliveryId
  }
  $webhook = Invoke-RestMethod -Method Post -Uri "http://localhost:$ApiPort/webhook/github" -Headers $webhookHeaders -Body $payload -ContentType "application/json"

  Assert-True ($webhook.ok -eq $true) "webhook response should be ok=true"

  Write-Host "[E2E 7/8] Verify events/alerts..." -ForegroundColor Cyan
  $events = Invoke-RestMethod -Method Get -Uri "http://localhost:$ApiPort/events?limit=20&offset=0&event_type=issues&action=opened" -Headers $authHeaders
  $alerts = Invoke-RestMethod -Method Get -Uri "http://localhost:$ApiPort/alerts?limit=20&offset=0&event_type=issues&action=opened&suggestion_type=label" -Headers $authHeaders

  Assert-True ($events.ok -eq $true) "events API should return ok=true"
  Assert-True ($alerts.ok -eq $true) "alerts API should return ok=true"
  Assert-True ($events.total -ge 1) "events total should be >= 1"
  Assert-True ($alerts.total -ge 1) "alerts total should be >= 1"

  $foundEvent = $false
  foreach ($e in $events.items) {
    if ($e.delivery_id -eq $deliveryId) { $foundEvent = $true; break }
  }
  Assert-True $foundEvent "should find event by delivery id"

  $foundAlert = $false
  foreach ($a in $alerts.items) {
    if ($a.delivery_id -eq $deliveryId -and $a.suggestion_value -eq "priority-high") { $foundAlert = $true; break }
  }
  Assert-True $foundAlert "should find alert by delivery id and suggestion value"

  Write-Host "[E2E 8/8] PASS" -ForegroundColor Green

  Write-Host ("events.total={0}, alerts.total={1}" -f $events.total, $alerts.total)
}
finally {
  if ($cleanupApi -and $null -ne $apiProc -and -not $apiProc.HasExited) {
    Stop-Process -Id $apiProc.Id -Force
    Write-Host "API process stopped: $($apiProc.Id)" -ForegroundColor Yellow
  }
  if ($cleanupWeb -and $null -ne $webProc -and -not $webProc.HasExited) {
    Stop-Process -Id $webProc.Id -Force
    Write-Host "Web process stopped: $($webProc.Id)" -ForegroundColor Yellow
  }
}
