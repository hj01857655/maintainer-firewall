param(
  [string]$AdminUsername = "admin",
  [string]$AdminPassword = "admin123",
  [string]$AccessToken = "mf-demo-token",
  [string]$GitHubWebhookSecret = "mf-demo-webhook-secret",
  [string]$GitHubToken = "",
  [string]$DatabaseURL = "postgres://postgres:postgres@localhost:5432/maintainer_firewall?sslmode=disable",
  [int]$ApiPort = 8080,
  [int]$WebPort = 5173
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$apiDir = Join-Path $repoRoot "apps\api-go"
$webDir = Join-Path $repoRoot "apps\web-react"

Write-Host "[1/7] Set environment variables..." -ForegroundColor Cyan
$env:ADMIN_USERNAME = $AdminUsername
$env:ADMIN_PASSWORD = $AdminPassword
$env:ACCESS_TOKEN = $AccessToken
$env:GITHUB_WEBHOOK_SECRET = $GitHubWebhookSecret
$env:GITHUB_TOKEN = $GitHubToken
$env:DATABASE_URL = $DatabaseURL
$env:PORT = "$ApiPort"

Write-Host "[2/7] Start API server in background..." -ForegroundColor Cyan
Push-Location $apiDir
$apiProc = Start-Process -FilePath "powershell" -ArgumentList "-NoProfile -Command go run .\cmd\server\main.go" -PassThru
Pop-Location

Write-Host "[3/7] Wait API health..." -ForegroundColor Cyan
$apiHealthUrl = "http://localhost:$ApiPort/health"
for ($i = 0; $i -lt 40; $i++) {
  try {
    $null = Invoke-RestMethod -Method Get -Uri $apiHealthUrl -TimeoutSec 2
    break
  } catch {
    Start-Sleep -Milliseconds 500
  }
  if ($i -eq 39) {
    throw "API did not become healthy in time."
  }
}

Write-Host "[4/7] Login and get access token..." -ForegroundColor Cyan
$loginResp = Invoke-RestMethod -Method Post -Uri "http://localhost:$ApiPort/auth/login" -ContentType "application/json" -Body (@{
  username = $AdminUsername
  password = $AdminPassword
} | ConvertTo-Json)
if (-not $loginResp.ok -or -not $loginResp.token) {
  throw "Login failed: $($loginResp | ConvertTo-Json -Compress)"
}
$bearer = $loginResp.token
$authHeaders = @{ Authorization = "Bearer $bearer" }

Write-Host "[5/7] Ensure demo rule exists..." -ForegroundColor Cyan
$ruleBody = @{
  event_type = "issues"
  keyword = "urgent"
  suggestion_type = "label"
  suggestion_value = "priority-high"
  reason = "demo urgent rule"
  is_active = $true
} | ConvertTo-Json
$null = Invoke-RestMethod -Method Post -Uri "http://localhost:$ApiPort/rules" -Headers $authHeaders -ContentType "application/json" -Body $ruleBody

Write-Host "[6/7] Send demo webhook..." -ForegroundColor Cyan
$deliveryId = "demo-" + [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
$body = '{"action":"opened","repository":{"full_name":"owner/repo"},"sender":{"login":"alice"},"issue":{"number":123,"title":"urgent bug report"}}'
$hmac = New-Object System.Security.Cryptography.HMACSHA256
$hmac.Key = [Text.Encoding]::UTF8.GetBytes($GitHubWebhookSecret)
$hashBytes = $hmac.ComputeHash([Text.Encoding]::UTF8.GetBytes($body))
$signature = "sha256=" + ([BitConverter]::ToString($hashBytes).Replace("-", "").ToLower())

$webhookResp = Invoke-RestMethod -Method Post -Uri "http://localhost:$ApiPort/webhook/github" -Headers @{
  "X-Hub-Signature-256" = $signature
  "X-GitHub-Event" = "issues"
  "X-GitHub-Delivery" = $deliveryId
} -Body $body -ContentType "application/json"

Write-Host "[7/7] Query events and alerts..." -ForegroundColor Cyan
$events = Invoke-RestMethod -Method Get -Uri "http://localhost:$ApiPort/events?limit=5&offset=0&event_type=issues&action=opened" -Headers $authHeaders
$alerts = Invoke-RestMethod -Method Get -Uri "http://localhost:$ApiPort/alerts?limit=5&offset=0&event_type=issues&action=opened&suggestion_type=label" -Headers $authHeaders

Write-Host "\n=== Demo Summary ===" -ForegroundColor Green
Write-Host ("API Health : {0}" -f $apiHealthUrl)
Write-Host ("Login OK   : token length {0}" -f ($bearer.Length))
Write-Host ("Webhook OK : {0}" -f ($webhookResp.ok))
Write-Host ("Events     : total={0}, showing={1}" -f $events.total, $events.items.Count)
Write-Host ("Alerts     : total={0}, showing={1}" -f $alerts.total, $alerts.items.Count)
Write-Host ("Web UI     : http://localhost:{0}" -f $WebPort)

Write-Host "\nTip: start web manually:" -ForegroundColor Yellow
Write-Host "  cd $webDir"
Write-Host "  npm install"
Write-Host "  npm run dev"

Write-Host "\nAPI process id: $($apiProc.Id)" -ForegroundColor Yellow
Write-Host "Stop API with: Stop-Process -Id $($apiProc.Id)" -ForegroundColor Yellow
