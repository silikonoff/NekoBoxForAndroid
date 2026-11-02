[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$SdkRoot,
    [Parameter(Mandatory)][string]$JavaHome,
    [string]$Go = 'go.exe',
    [switch]$SkipLibcore
)
$ErrorActionPreference = 'Stop'
$repo = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..'))
$env:JAVA_HOME = (Resolve-Path -LiteralPath $JavaHome).Path
$env:ANDROID_HOME = (Resolve-Path -LiteralPath $SdkRoot).Path
$env:ANDROID_USER_HOME = Join-Path $repo '.build\android'
$env:GRADLE_USER_HOME = Join-Path $repo '.build\gradle'
$env:Path = "$env:JAVA_HOME\bin;$env:Path"
if (!$SkipLibcore) { & (Join-Path $PSScriptRoot 'Build-Libcore.ps1') -SdkRoot $SdkRoot -JavaHome $JavaHome -Go $Go }
if (!(Test-Path -LiteralPath (Join-Path $repo 'app\libs\libcore.aar'))) { throw 'Build libcore AAR first' }
& (Join-Path $PSScriptRoot 'Prepare-Assets.ps1')
$properties = Join-Path $repo 'local.properties'
$lines = if (Test-Path -LiteralPath $properties) { @(Get-Content -LiteralPath $properties | Where-Object { $_ -notmatch '^sdk\.dir\s*=' }) } else { @() }
$lines += 'sdk.dir=' + $env:ANDROID_HOME.Replace('\','/')
Set-Content -LiteralPath $properties -Value $lines -Encoding utf8
& (Join-Path $repo 'gradlew.bat') -p $repo --no-daemon --max-workers=2 '-Pkotlin.compiler.execution.strategy=in-process' app:assembleOssDebug
if ($LASTEXITCODE) { throw 'Android APK build failed' }
Get-ChildItem (Join-Path $repo 'app\build\outputs\apk\oss\debug') -Filter '*arm64-v8a*.apk' | Get-FileHash -Algorithm SHA256
