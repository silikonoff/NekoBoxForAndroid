[CmdletBinding()]
param()
$ErrorActionPreference = 'Stop'
$repo = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..'))
$cache = Join-Path $repo '.build\assets'
$archive = Join-Path $cache 'upstream-1.4.2.apk'
$expected = 'd01e76feeb05e4701f3ae382cb71c0a3ebb77039005e5893f8817e1051f8ac28'
New-Item -ItemType Directory -Force $cache | Out-Null
if (!(Test-Path -LiteralPath $archive) -or (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant() -ne $expected) {
    Invoke-WebRequest 'https://github.com/MatsuriDayo/NekoBoxForAndroid/releases/download/1.4.2/NekoBox-1.4.2-arm64-v8a.apk' -OutFile $archive
}
if ((Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant() -ne $expected) { throw 'Upstream asset archive checksum mismatch' }
$files = @('assets/sing-box/geoip.db.xz','assets/sing-box/geoip.version.txt','assets/sing-box/geosite.db.xz','assets/sing-box/geosite.version.txt','assets/yacd.version.txt','assets/yacd.zip')
$zip = [IO.Compression.ZipFile]::OpenRead($archive)
try {
    foreach ($name in $files) {
        $entry = $zip.GetEntry($name)
        if (!$entry) { throw "Missing pinned asset: $name" }
        $target = Join-Path $repo ('app/src/main/' + $name)
        if (Test-Path -LiteralPath $target) { continue } # Preserve local/custom assets.
        New-Item -ItemType Directory -Force (Split-Path $target) | Out-Null
        [IO.Compression.ZipFileExtensions]::ExtractToFile($entry, $target, $false)
    }
} finally { $zip.Dispose() }
Write-Output 'Pinned GeoIP, Geosite and Yacd assets prepared; existing assets preserved.'
