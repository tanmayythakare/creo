$ErrorActionPreference = "Continue"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
$creo    = Join-Path $ProjectRoot "creo.exe"
$base    = Join-Path $ScriptDir "brutal_tests"
$answers = "y`ndev`n"

function Test-Creo {
    param([string]$Label, [string]$Dir)

    Remove-Item "$Dir\creo.yml" -ErrorAction SilentlyContinue

    Push-Location $Dir
    $raw = ($answers | & $creo init --force 2>&1)
    Pop-Location

    # Strip non-ASCII / box-drawing chars, keep printable ASCII
    $clean = ($raw | ForEach-Object { $_ -replace '[^\x20-\x7E]', '' }) -join "`n"

    $lines = $clean -split "`n"
    $result = $lines | Where-Object {
        ($_ -match 'found \d+ service') -or
        ($_ -match '^\s+\S') -or
        ($_ -match 'creo\.yml created') -or
        ($_ -match 'cannot create config') -or
        ($_ -match 'Error:') -or
        ($_ -match 'looks right') -or
        ($_ -match 'workflow name')
    } | Where-Object { $_.Trim() -ne '' } | Select-Object -First 10

    Write-Host ""
    Write-Host "[$Label]" -ForegroundColor Yellow
    $result | ForEach-Object { Write-Host "  $_" }
}

# ── Setup ─────────────────────────────────────────────────────────────────────
Remove-Item -Recurse -Force $base -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $base | Out-Null

# Test 1: Next.js only
$d = "$base\test_nextjs"
New-Item -ItemType Directory -Force -Path $d | Out-Null
'{"dependencies":{"next":"13.0.0","react":"18.0.0"}}' | Set-Content "$d\package.json"

# Test 2: NestJS backend only
$d = "$base\test_nestjs"
New-Item -ItemType Directory -Force -Path $d | Out-Null
'{}' | Set-Content "$d\nest-cli.json"
'{"dependencies":{"@nestjs/core":"10.0.0"}}' | Set-Content "$d\package.json"

# Test 3: Django
$d = "$base\test_django"
New-Item -ItemType Directory -Force -Path $d | Out-Null
'' | Set-Content "$d\manage.py"
'django>=4.0' | Set-Content "$d\requirements.txt"

# Test 4: Full stack (Next.js + Spring Boot + Docker)
$d = "$base\test_fullstack"
New-Item -ItemType Directory -Force -Path "$d\frontend" | Out-Null
New-Item -ItemType Directory -Force -Path "$d\backend"  | Out-Null
New-Item -ItemType Directory -Force -Path "$d\database" | Out-Null
'{"dependencies":{"next":"13.0.0"}}' | Set-Content "$d\frontend\package.json"
'<project></project>'                | Set-Content "$d\backend\pom.xml"
"services:`n  postgres:`n    image: postgres" | Set-Content "$d\database\docker-compose.yml"

# Test 5: Monorepo with turbo
$d = "$base\test_monorepo"
New-Item -ItemType Directory -Force -Path $d | Out-Null
'{"pipeline":{}}' | Set-Content "$d\turbo.json"
'{"dependencies":{"next":"13.0.0"}}' | Set-Content "$d\package.json"

# Test 6: Laravel + Vue
$d = "$base\test_laravel"
New-Item -ItemType Directory -Force -Path $d | Out-Null
'' | Set-Content "$d\artisan"
'{"dependencies":{"vue":"3.0.0"}}' | Set-Content "$d\package.json"

# Test 7: Flutter
$d = "$base\test_flutter"
New-Item -ItemType Directory -Force -Path $d | Out-Null
'name: myapp' | Set-Content "$d\pubspec.yaml"

# Test 8: .NET
$d = "$base\test_dotnet"
New-Item -ItemType Directory -Force -Path $d | Out-Null
'' | Set-Content "$d\MyApp.csproj"

# Test 9: Malformed package.json
$d = "$base\test_malformed"
New-Item -ItemType Directory -Force -Path $d | Out-Null
'{broken json here' | Set-Content "$d\package.json"

# Test 10: Empty directory
$d = "$base\test_empty"
New-Item -ItemType Directory -Force -Path $d | Out-Null

# ── Run ───────────────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "   CREO BRUTAL TEST RESULTS — 10 CASES    " -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

Test-Creo "TEST 1 - Next.js only"                                "$base\test_nextjs"
Test-Creo "TEST 2 - NestJS backend only"                         "$base\test_nestjs"
Test-Creo "TEST 3 - Django project"                              "$base\test_django"
Test-Creo "TEST 4 - Full stack Next.js plus Spring Boot plus Docker" "$base\test_fullstack"
Test-Creo "TEST 5 - Monorepo with turbo"                         "$base\test_monorepo"
Test-Creo "TEST 6 - Laravel + Vue"                               "$base\test_laravel"
Test-Creo "TEST 7 - Flutter mobile"                              "$base\test_flutter"
Test-Creo "TEST 8 - .NET project"                                "$base\test_dotnet"
Test-Creo "TEST 9 - Malformed package.json"                      "$base\test_malformed"
Test-Creo "TEST 10 - Empty directory"                            "$base\test_empty"

Write-Host ""
Write-Host "==========================================" -ForegroundColor Green
Write-Host "   ALL 10 BRUTAL TESTS COMPLETE           " -ForegroundColor Green
Write-Host "==========================================" -ForegroundColor Green
Write-Host ""

# ── Cleanup ───────────────────────────────────────────────────────────────────
Remove-Item -Recurse -Force $base -ErrorAction SilentlyContinue
Write-Host "Test directories cleaned up." -ForegroundColor Gray
