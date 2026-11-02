[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$SdkRoot,
    [Parameter(Mandatory)][string]$JavaHome,
    [string]$Go = 'go.exe',
    [string]$Target = 'android/arm64',
    [switch]$PrepareOnly
)
$ErrorActionPreference = 'Stop'
$repo = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..'))
$dependencies = Split-Path $repo
$cache = Join-Path $repo '.build\windows'
$env:JAVA_HOME = (Resolve-Path -LiteralPath $JavaHome).Path
$env:ANDROID_HOME = (Resolve-Path -LiteralPath $SdkRoot).Path
$env:ANDROID_SDK_ROOT = $env:ANDROID_HOME
if (!$env:ANDROID_USER_HOME) { $env:ANDROID_USER_HOME = Join-Path $cache 'android' }
$env:GOPATH = Join-Path $cache 'gopath'
$env:GOCACHE = Join-Path $cache 'go-cache'
$env:GOTMPDIR = Join-Path $cache 'tmp'
$env:GOMAXPROCS = '2'
$env:GOFLAGS = '-p=2'
New-Item -ItemType Directory -Force $env:GOTMPDIR | Out-Null
$env:TEMP = $env:GOTMPDIR
$env:TMP = $env:GOTMPDIR
$goExe = (Get-Command $Go -ErrorAction Stop).Source
$env:Path = "$(Split-Path $goExe);$env:JAVA_HOME\bin;$env:GOPATH\bin;$env:Path"

function Invoke-Git([string[]]$Arguments) {
    & git.exe @Arguments
    if ($LASTEXITCODE -ne 0) { throw "git failed ($LASTEXITCODE): $($Arguments[0])" }
}
function Prepare-Source([string]$Name, [string]$Url, [string]$Commit) {
    $path = Join-Path $dependencies $Name
    if (!(Test-Path -LiteralPath $path)) {
        Invoke-Git -Arguments @('clone','--no-checkout',$Url,$path)
    }
    $head = & git.exe -C $path rev-parse HEAD 2>$null
    if ($head -ne $Commit) {
        $dirty = & git.exe -C $path status --porcelain
        if ($dirty) { throw "Preserving modified dependency: $path" }
        Invoke-Git -Arguments @('-C',$path,'fetch','origin',$Commit)
        Invoke-Git -Arguments @('-C',$path,'checkout','--detach',$Commit)
    }
    return $path
}
$pins = @{}
foreach ($line in Get-Content -LiteralPath (Join-Path $repo 'buildScript\lib\core\get_source_env.sh')) {
    if ($line -match '^export (COMMIT_[A-Z_]+)="([0-9a-f]{40})"$') { $pins[$Matches[1]] = $Matches[2] }
}
foreach ($pin in 'COMMIT_SING_BOX','COMMIT_LIBNEKO','COMMIT_GOMOBILE') {
    if (!$pins[$pin]) { throw "Missing source pin: $pin" }
}
$core = Prepare-Source 'sing-box-1.14.4' 'https://github.com/SagerNet/sing-box.git' $pins.COMMIT_SING_BOX
$null = Prepare-Source 'libneko' 'https://github.com/MatsuriDayo/libneko.git' $pins.COMMIT_LIBNEKO
$mobile = Prepare-Source 'gomobile' 'https://github.com/MatsuriDayo/gomobile.git' $pins.COMMIT_GOMOBILE
$patch = Join-Path $repo 'buildScript\patches\nekobox-core.patch'
& git.exe -C $core apply --reverse --check $patch 2>$null
if ($LASTEXITCODE -ne 0) {
    Invoke-Git -Arguments @('-C',$core,'apply','--check',$patch)
    Invoke-Git -Arguments @('-C',$core,'apply',$patch)
}
if ($PrepareOnly) { return }
& $goExe -C $mobile install ./cmd/gomobile ./cmd/gobind
if ($LASTEXITCODE) { throw 'gomobile tool build failed' }
$env:GOBIND = 'gobind'
$gomobile = Join-Path $env:GOPATH 'bin\gomobile.exe'
& $gomobile init
if ($LASTEXITCODE) { throw 'gomobile init failed' }
$output = Join-Path $repo 'app\libs\libcore.aar'
New-Item -ItemType Directory -Force (Split-Path $output) | Out-Null
Push-Location (Join-Path $repo 'libcore')
try {
    & $gomobile bind -target $Target -androidapi 21 -o $output -trimpath -ldflags '-s -w -X github.com/sagernet/sing-box/constant.Version=1.14.4-neko.3' -tags 'with_gvisor,with_quic,with_wireguard,with_utls,with_clash_api' .
    if ($LASTEXITCODE) { throw 'libcore AAR build failed' }
} finally { Pop-Location }
Get-FileHash -LiteralPath $output -Algorithm SHA256
