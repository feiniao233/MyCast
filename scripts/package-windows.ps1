#requires -Version 7.0
param([string]$GoExe='go',[string]$Version='0.1.0-alpha-dev',[switch]$LayoutOnly)
$ErrorActionPreference='Stop'
if($Version.Length-gt 64 -or $Version-notmatch '^\d+\.\d+\.\d+-alpha(?:[.-][a-zA-Z0-9]+)*$'){throw 'Use a bounded alpha version label'}
$taskRepo=[IO.Path]::GetFullPath((Split-Path $PSScriptRoot -Parent))
Set-Location -LiteralPath $taskRepo
if ((git status --porcelain)) {throw 'Packaging requires a clean committed tree'}
$taskCommit=(& git rev-parse HEAD).Trim()
$taskTree=(& git rev-parse 'HEAD^{tree}').Trim()
$taskShort=$taskCommit.Substring(0,12)
$taskGoVersion=(& $GoExe version | Out-String).Trim()
if($LASTEXITCODE-ne 0 -or $taskGoVersion -notmatch '^go version go1\.27\.2 '){throw 'Use official Go 1.27.2 for packaging'}
$taskBinary=Join-Path $taskRepo 'dist\MyCast.exe'
$taskMetadata=(& $GoExe version -m 'dist/MyCast.exe' | Out-String)
if($LASTEXITCODE-ne 0 -or $taskMetadata -notmatch '(?m)^.+:\s+go1\.27\.2\s*$' -or $taskMetadata -notmatch "vcs.revision=$taskCommit" -or $taskMetadata -notmatch 'vcs.modified=false' -or $taskMetadata -notmatch 'CGO_ENABLED=0' -or $taskMetadata -notmatch 'GOOS=windows' -or $taskMetadata -notmatch 'GOARCH=amd64' -or $taskMetadata -notmatch 'mygo_noinspector' -or $taskMetadata -notmatch 'github.com/egoist/mygo\s+v0\.4\.6'){throw 'Binary metadata must match the clean Windows amd64 commit and locked runtime'}
if($taskMetadata -notmatch '(?m)^dist[\\/]MyCast\.exe:\s+go1\.27\.2\s*$' -or $taskMetadata.Contains($taskRepo)){throw 'Build info must use a relative binary name without this checkout path'}
& $GoExe run ./scripts/verify-icon -exe $taskBinary | Out-Host
if($LASTEXITCODE-ne 0){throw 'Packaged executable icon differs from canonical source ICO'}
$taskOutput=Join-Path $taskRepo 'dist'
$taskStage=Join-Path $taskOutput ('package-stage-'+[guid]::NewGuid().ToString('N'))
$taskPackage=Join-Path $taskStage 'package'
$taskSource=Join-Path $taskStage 'source'
New-Item -ItemType Directory -Path $taskPackage,$taskSource | Out-Null
Copy-Item -LiteralPath $taskBinary -Destination (Join-Path $taskPackage 'MyCast.exe')
Copy-Item -LiteralPath README.md,LICENSE,THIRD_PARTY_NOTICES.md,AGENTS.md -Destination $taskPackage
Copy-Item -LiteralPath docs,licenses -Destination $taskPackage -Recurse
[IO.File]::WriteAllText((Join-Path $taskPackage 'BUILDINFO.txt'),$taskMetadata,[Text.UTF8Encoding]::new($false))
$taskSourceArchive=Join-Path $taskStage 'tracked-source.zip'
git archive --format=zip HEAD -o $taskSourceArchive
if($LASTEXITCODE-ne 0){throw 'Source archive failed'}
Expand-Archive -LiteralPath $taskSourceArchive -DestinationPath $taskSource
Push-Location -LiteralPath $taskSource
try {
  & $GoExe mod vendor
  if($LASTEXITCODE-ne 0){throw 'Vendored corresponding source preparation failed'}
} finally {Pop-Location}
$taskSourceZip=Join-Path $taskPackage "MyCast-source-$taskShort.zip"
Add-Type -AssemblyName System.IO.Compression.FileSystem
[IO.Compression.ZipFile]::CreateFromDirectory($taskSource,$taskSourceZip,[IO.Compression.CompressionLevel]::Optimal,$false)
$taskFiles=Get-ChildItem -LiteralPath $taskPackage -File -Recurse -Force | Sort-Object FullName
$taskHashes=@($taskFiles | ForEach-Object {[pscustomobject]@{path=[IO.Path]::GetRelativePath($taskPackage,$_.FullName).Replace('\','/');sha256=(Get-FileHash -Algorithm SHA256 -LiteralPath $_.FullName).Hash.ToLowerInvariant();bytes=$_.Length}})
$taskManifest=[ordered]@{version=$Version;status='development-preview';commit=$taskCommit;tree=$taskTree;go='1.27.2';mygo='v0.4.6';os='windows';arch='amd64';cgo=$false;nativeDesktopAcceptance='not-tested';sourceArchive=[IO.Path]::GetFileName($taskSourceZip);vendoredDependencies=$true;files=$taskHashes}
[IO.File]::WriteAllText((Join-Path $taskPackage 'MANIFEST.json'),($taskManifest | ConvertTo-Json -Depth 8),[Text.UTF8Encoding]::new($false))
$taskSumFiles=Get-ChildItem -LiteralPath $taskPackage -File -Recurse -Force | Sort-Object FullName
$taskSumLines=@($taskSumFiles | ForEach-Object {(Get-FileHash -Algorithm SHA256 -LiteralPath $_.FullName).Hash.ToLowerInvariant()+'  '+[IO.Path]::GetRelativePath($taskPackage,$_.FullName).Replace('\','/')})
[IO.File]::WriteAllLines((Join-Path $taskPackage 'SHA256SUMS.txt'),$taskSumLines,[Text.UTF8Encoding]::new($false))
if($env:GITHUB_OUTPUT){Add-Content -LiteralPath $env:GITHUB_OUTPUT -Value "package_path=$taskPackage"}
if($LayoutOnly){[pscustomobject]@{packagePath=$taskPackage;commit=$taskCommit;sourceVendored=$true}|ConvertTo-Json;exit}
$taskZip=Join-Path $taskOutput "MyCast-windows-amd64-$Version-$taskShort.zip"
[IO.Compression.ZipFile]::CreateFromDirectory($taskPackage,$taskZip,[IO.Compression.CompressionLevel]::Optimal,$false)
$taskZipHash=(Get-FileHash -Algorithm SHA256 -LiteralPath $taskZip).Hash.ToLowerInvariant()
[IO.File]::WriteAllText(($taskZip+'.sha256.txt'),$taskZipHash+'  '+[IO.Path]::GetFileName($taskZip)+[Environment]::NewLine,[Text.UTF8Encoding]::new($false))
[pscustomobject]@{packagePath=$taskPackage;zip=$taskZip;sha256=$taskZipHash;commit=$taskCommit;sourceVendored=$true;nativeDesktopAcceptance='not-tested'}|ConvertTo-Json
