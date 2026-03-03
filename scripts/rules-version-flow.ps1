param(
  [string]$AdminUsername = "admin",
  [string]$AdminPassword = "CHANGE_ME_ADMIN_PASSWORD",
  [string]$TenantID = "default",
  [int]$ApiPort = 8080
)

$ErrorActionPreference = "Stop"

function Assert-True {
  param([bool]$Condition, [string]$Message)
  if (-not $Condition) {
    throw "[ASSERT FAILED] $Message"
  }
}

function New-RuleBody {
  param([string]$Keyword, [string]$SuggestionValue)
  return @{
    event_type = "issues"
    keyword = $Keyword
    suggestion_type = "label"
    suggestion_value = $SuggestionValue
    reason = "rules version flow test"
    is_active = $true
  } | ConvertTo-Json
}

$baseUrl = "http://localhost:$ApiPort"

Write-Host "[1/8] Login..." -ForegroundColor Cyan
$loginBody = @{
  username = $AdminUsername
  password = $AdminPassword
  tenant_id = $TenantID
} | ConvertTo-Json
$login = Invoke-RestMethod -Method Post -Uri "$baseUrl/auth/login" -ContentType "application/json" -Body $loginBody
Assert-True ($login.ok -eq $true) "login should return ok=true"
Assert-True (-not [string]::IsNullOrWhiteSpace($login.token)) "login token should not be empty"
$authHeaders = @{ Authorization = "Bearer $($login.token)" }

Write-Host "[2/8] Create baseline rule..." -ForegroundColor Cyan
$null = Invoke-RestMethod -Method Post -Uri "$baseUrl/api/rules" -Headers $authHeaders -ContentType "application/json" -Body (New-RuleBody -Keyword "version-flow-v1" -SuggestionValue "priority-v1")

Write-Host "[3/8] Publish v1..." -ForegroundColor Cyan
$publish1 = Invoke-RestMethod -Method Post -Uri "$baseUrl/api/rules/publish" -Headers $authHeaders
Assert-True ($publish1.ok -eq $true) "publish v1 should return ok=true"
Assert-True ($publish1.version -gt 0) "publish v1 should return version > 0"
$v1 = [int64]$publish1.version

Write-Host "[4/8] Create changed rule and publish v2..." -ForegroundColor Cyan
$null = Invoke-RestMethod -Method Post -Uri "$baseUrl/api/rules" -Headers $authHeaders -ContentType "application/json" -Body (New-RuleBody -Keyword "version-flow-v2" -SuggestionValue "priority-v2")
$publish2 = Invoke-RestMethod -Method Post -Uri "$baseUrl/api/rules/publish" -Headers $authHeaders
Assert-True ($publish2.ok -eq $true) "publish v2 should return ok=true"
Assert-True ([int64]$publish2.version -gt $v1) "publish v2 should be newer than v1"

Write-Host "[5/8] List versions..." -ForegroundColor Cyan
$versions = Invoke-RestMethod -Method Get -Uri "$baseUrl/api/rules/versions?limit=20&offset=0" -Headers $authHeaders
Assert-True ($versions.ok -eq $true) "list versions should return ok=true"
Assert-True ($versions.total -ge 2) "versions total should be >= 2"

Write-Host "[6/8] Replay by historical version..." -ForegroundColor Cyan
$replayV1Body = @{
  version = $v1
  event_type = "issues"
  payload = @{
    issue = @{
      title = "version-flow-v1 test title"
      body = "rules replay validation"
    }
  }
} | ConvertTo-Json -Depth 6
$replayV1 = Invoke-RestMethod -Method Post -Uri "$baseUrl/api/rules/replay" -Headers $authHeaders -ContentType "application/json" -Body $replayV1Body
Assert-True ($replayV1.ok -eq $true) "replay by version should return ok=true"
Assert-True ($replayV1.version -eq $v1) "replay response version should equal requested version"

Write-Host "[7/8] Rollback to v1..." -ForegroundColor Cyan
$rollbackHeaders = @{
  Authorization = "Bearer $($login.token)"
  "X-MF-Confirm" = "confirm"
}
$rollbackBody = @{ version = $v1 } | ConvertTo-Json
$rollback = Invoke-RestMethod -Method Post -Uri "$baseUrl/api/rules/rollback" -Headers $rollbackHeaders -ContentType "application/json" -Body $rollbackBody
Assert-True ($rollback.ok -eq $true) "rollback should return ok=true"
Assert-True ($rollback.from_version -eq $v1) "rollback from_version should equal target version"
Assert-True ($rollback.to_version -gt $v1) "rollback should create a new version snapshot"

Write-Host "[8/8] Replay current active rules (version=0)..." -ForegroundColor Cyan
$replayCurrentBody = @{
  version = 0
  event_type = "issues"
  payload = @{
    issue = @{
      title = "version-flow-v1 current replay title"
      body = "current active rules replay validation"
    }
  }
} | ConvertTo-Json -Depth 6
$replayCurrent = Invoke-RestMethod -Method Post -Uri "$baseUrl/api/rules/replay" -Headers $authHeaders -ContentType "application/json" -Body $replayCurrentBody
Assert-True ($replayCurrent.ok -eq $true) "replay current should return ok=true"
Assert-True ($replayCurrent.version -eq 0) "replay current should keep version=0 in response"

Write-Host "`nPASS: rules version flow verified." -ForegroundColor Green
Write-Host ("v1={0}, latest publish={1}, rollback_to={2}" -f $v1, $publish2.version, $rollback.to_version)
