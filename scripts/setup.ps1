# scripts/setup.ps1 — StageSync 개발 환경 일괄 설치 (Windows)
#
# 대상: Windows 10 1809+ / Windows 11
# 전제: PowerShell 5.1+ 또는 PowerShell 7
# 실행:
#   일반 (관리자 아님) PowerShell 에서:
#     Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
#     .\scripts\setup.ps1
#
# idempotent — 여러 번 실행 OK, 이미 설치된 건 스킵.
#
# 설치 범위 (Phase 0 ~ 7 필요 도구 일괄):
#   - scoop 패키지 매니저 자체                           [자동 설치]
#   - go, protobuf(protoc), sqlc, goose, golangci-lint,
#     make                                              [scoop]
#   - protoc-gen-go                                     [go install]
#   - Docker Desktop                                    [winget — scoop 에 GUI 앱 없음]
#   - mysql:8 · redis:7-alpine 이미지 pre-pull          [docker — Desktop 실행 중일 때만]
# Phase 14+ 도구 (kubectl, terraform, k6, locust) 는 해당 Phase 진입 시 추가 예정.

$ErrorActionPreference = 'Stop'

# 콘솔 출력 인코딩 UTF-8 로 고정 (한글 깨짐 방지 — Windows 기본 CP949)
try {
    [Console]::OutputEncoding = [System.Text.Encoding]::UTF8
    $OutputEncoding = [System.Text.Encoding]::UTF8
    chcp 65001 *> $null
} catch { }

# ---------- 색상 로그 ----------
function Info($msg) { Write-Host "[INFO] $msg" -ForegroundColor Blue }
function Ok($msg)   { Write-Host "[OK]   $msg" -ForegroundColor Green }
function Skip($msg) { Write-Host "[SKIP] $msg" -ForegroundColor Yellow }
function Err($msg)  { Write-Host "[ERR]  $msg" -ForegroundColor Red }

# ---------- 헬퍼 ----------
function Ensure-Scoop {
    if (Get-Command scoop -ErrorAction SilentlyContinue) {
        Skip "scoop (이미 설치됨)"
        return
    }
    Info "scoop 설치 중 (https://get.scoop.sh)"
    # 현재 유효 정책이 이미 실행 가능하면 건드리지 않음 (상위 스코프가 Bypass 인 경우 등)
    $effective = Get-ExecutionPolicy
    if ($effective -in @('Restricted', 'AllSigned', 'Undefined')) {
        try {
            Set-ExecutionPolicy -Scope CurrentUser RemoteSigned -Force -ErrorAction Stop
        } catch {
            Info "ExecutionPolicy 변경 실패 (상위 스코프가 재정의 중) — 현재 유효 정책 '$effective' 로 진행"
        }
    }
    Invoke-RestMethod -Uri 'https://get.scoop.sh' | Invoke-Expression
    # 현재 세션에 scoop PATH 반영
    $env:PATH = "$env:USERPROFILE\scoop\shims;$env:PATH"
    if (Get-Command scoop -ErrorAction SilentlyContinue) {
        Ok 'scoop'
    } else {
        Err 'scoop 설치 실패 — 수동 설치 후 재실행 필요'
        exit 1
    }
}

function Scoop-Pkg {
    param([string]$Pkg)
    $installed = scoop list $Pkg 2>$null | Select-String -SimpleMatch $Pkg
    if ($installed) {
        Skip "$Pkg (scoop, 이미 설치됨)"
    } else {
        Info "scoop install $Pkg"
        scoop install $Pkg
        if ($LASTEXITCODE -eq 0) { Ok $Pkg }
    }
}

function Go-Cli {
    param([string]$Module)
    $bin = ([IO.Path]::GetFileName(($Module -split '@')[0]))
    if (Get-Command $bin -ErrorAction SilentlyContinue) {
        Skip "$bin (go install, 이미 있음)"
    } else {
        Info "go install $Module"
        go install $Module
        if ($LASTEXITCODE -eq 0) { Ok $bin }
    }
}

function Winget-Pkg {
    param([string]$Id, [string]$DisplayName = $null)
    if (-not $DisplayName) { $DisplayName = $Id }
    if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
        Skip "$DisplayName — winget 미설치, 수동 설치 필요 (MS Store 에서 'App Installer' 업데이트)"
        return
    }
    $installed = winget list --id $Id --exact --accept-source-agreements 2>$null | Select-String -SimpleMatch $Id
    if ($installed) {
        Skip "$DisplayName (winget, 이미 설치됨)"
    } else {
        Info "winget install $Id"
        winget install --id $Id --exact --silent --accept-package-agreements --accept-source-agreements
        if ($LASTEXITCODE -eq 0) { Ok $DisplayName }
    }
}

function Docker-Pull {
    param([string]$Image)
    docker image inspect $Image *> $null
    if ($LASTEXITCODE -eq 0) {
        Skip "$Image (이미 로컬에 존재)"
    } else {
        Info "docker pull $Image"
        docker pull $Image
        if ($LASTEXITCODE -eq 0) { Ok $Image }
    }
}

# ---------- 설치 실행 ----------
Write-Host "======================================================"
Write-Host "  StageSync 개발 환경 설치 — Windows (Phase 0-7 일괄)"
Write-Host "======================================================"
Write-Host ""

Info "[1/5] scoop 패키지 매니저"
Ensure-Scoop

Info "[2/5] Go · 빌드 도구 (scoop)"
Scoop-Pkg 'go'
Scoop-Pkg 'protobuf'
Scoop-Pkg 'sqlc'
Scoop-Pkg 'goose'
Scoop-Pkg 'golangci-lint'
Scoop-Pkg 'make'

# Go 가 방금 설치된 경우 GOPATH/bin 세션 PATH 에 반영
$goPath = & go env GOPATH 2>$null
if ($goPath) {
    $env:PATH = "$goPath\bin;$env:PATH"
}

Info "[3/5] Go 기반 플러그인 (go install)"
Go-Cli 'google.golang.org/protobuf/cmd/protoc-gen-go@latest'

Info "[4/5] Docker Desktop (winget — scoop 에 GUI 앱 없음)"
Winget-Pkg -Id 'Docker.DockerDesktop' -DisplayName 'Docker Desktop'
Info "Docker Desktop 은 첫 실행 → WSL2 backend 활성화 → 로그인 후 사용 가능."

Info "[5/5] Docker 이미지 pre-pull (MySQL · Redis)"
$dockerReady = $false
if (Get-Command docker -ErrorAction SilentlyContinue) {
    docker info *> $null
    if ($LASTEXITCODE -eq 0) { $dockerReady = $true }
}
if ($dockerReady) {
    Docker-Pull 'mysql:8'
    Docker-Pull 'redis:7-alpine'
} else {
    Skip "Docker daemon 미동작 — 이미지 pull 생략 (Docker Desktop 실행 후 재시도 또는 'make dev-up' 시 자동 pull)"
}

# ---------- PATH 영구 설정 안내 ----------
Write-Host ""
if ($goPath) {
    $goBin = "$goPath\bin"
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ($userPath -notlike "*$goBin*") {
        Info "Go bin 을 User PATH 에 영구 추가하려면:"
        Write-Host ""
        Write-Host "  [Environment]::SetEnvironmentVariable('Path', `"`$([Environment]::GetEnvironmentVariable('Path','User'));$goBin`", 'User')"
        Write-Host ""
        Info "scoop 의 shims 는 설치 과정에서 자동 등록되었습니다 (go · protoc · make 등)."
    }
}

# ---------- 설치 버전 요약 ----------
Write-Host ""
Info "설치된 도구 버전"
Write-Host "------------------------------------------------------"

function Show-Version {
    param([string]$Name, [scriptblock]$Cmd)
    try {
        $out = & $Cmd 2>&1 | Select-Object -First 1
        "{0,-18} {1}" -f $Name, $out
    } catch {
        "{0,-18} (미설치 또는 실행 실패)" -f $Name
    }
}

Show-Version 'go'             { go version }
Show-Version 'protoc'         { protoc --version }
Show-Version 'protoc-gen-go'  { protoc-gen-go --version }
Show-Version 'sqlc'           { sqlc version }
Show-Version 'goose'          { goose -version }
Show-Version 'golangci-lint'  { golangci-lint --version }
Show-Version 'make'           { make --version }
Show-Version 'docker'         { docker --version }
Show-Version 'docker compose' { docker compose version }

Write-Host ""
Ok "모든 필수 도구 설치 시도 완료!"
Write-Host ""
Write-Host "개발 워크플로우 (Windows — Docker Desktop backend):"
Write-Host "  준비:  Docker Desktop 실행 (트레이 아이콘 초록 확인)"
Write-Host "  시작:  make dev-up       # MySQL + Redis 컨테이너 기동"
Write-Host "  실행:  make run-mysql    # 서버 기동 (MYSQL_DSN 자동 세팅)"
Write-Host "  검증:  curl.exe http://localhost:5050/api/profile/p1"
Write-Host "  종료:  make dev-down     # MySQL + Redis 컨테이너 정리"
Write-Host ""
Write-Host "개별 제어:"
Write-Host "  make mysql-dev / mysql-stop / redis-dev / redis-stop"
Write-Host ""
