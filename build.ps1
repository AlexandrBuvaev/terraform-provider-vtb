# Параметры сборки
$HOSTNAME = "vtb"
$NAMESPACE = "vtb-cloud"
$NAME = "vtb"
$BINARY = "terraform-provider-${NAME}"
$VERSION = "2.16.0"
$OS_ARCH = "windows_amd64"

# Аналог grep для PowerShell
function Get-GoPackages {
    go list ./... | Where-Object { $_ -notmatch 'vendor' }
}

# Основные команды
function Build {
    $env:CGO_ENABLED = "0"
    go build -o "${BINARY}.exe"
    if (-not (Test-Path "${BINARY}.exe")) {
        throw "Build error: file ${BINARY}.exe does not build"
    }
}

function Release {
    ReleaseWindows
    ReleaseLinux
    ReleaseMacosM1
    ReleaseMacosIntel
}

function ReleaseLinux {
    $env:CGO_ENABLED = "0"
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    go build -o "bin\${BINARY}_${VERSION}_linux_amd64"
    if (Test-Path "bin\${BINARY}_${VERSION}_linux_amd64") {
        Compress-Archive -Path "bin\${BINARY}_${VERSION}_linux_amd64" -DestinationPath "bin\${BINARY}_${VERSION}_linux_amd64.zip" -Force
    }
}

function ReleaseWindows {
    $env:CGO_ENABLED = "0"
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    go build -o "bin\${BINARY}_${VERSION}_windows_amd64.exe"
    if (Test-Path "bin\${BINARY}_${VERSION}_windows_amd64.exe") {
        Compress-Archive -Path "bin\${BINARY}_${VERSION}_windows_amd64.exe" -DestinationPath "bin\${BINARY}_${VERSION}_windows_amd64.zip" -Force
    }
}

function ReleaseMacosM1 {
    $env:CGO_ENABLED = "0"
    $env:GOOS = "darwin"
    $env:GOARCH = "arm64"
    New-Item -ItemType Directory -Path "bin\macos" -Force | Out-Null
    go build -o "bin\macos\${BINARY}_${VERSION}_darwin_arm64"
    if (Test-Path "bin\macos\${BINARY}_${VERSION}_darwin_arm64") {
        Compress-Archive -Path "bin\macos\${BINARY}_${VERSION}_darwin_arm64" -DestinationPath "bin\macos\${BINARY}_${VERSION}_darwin_arm64.zip" -Force
    }
}

function ReleaseMacosIntel {
    $env:CGO_ENABLED = "0"
    $env:GOOS = "darwin"
    $env:GOARCH = "amd64"
    New-Item -ItemType Directory -Path "bin\macos" -Force | Out-Null
    go build -o "bin\macos\${BINARY}_${VERSION}_darwin_amd64"
    if (Test-Path "bin\macos\${BINARY}_${VERSION}_darwin_amd64") {
        Compress-Archive -Path "bin\macos\${BINARY}_${VERSION}_darwin_amd64" -DestinationPath "bin\macos\${BINARY}_${VERSION}_darwin_amd64.zip" -Force
    }
}

function InstallLocal {
    $targetDir = "$env:USERPROFILE\AppData\Roaming\terraform.d\plugins\${HOSTNAME}\${NAMESPACE}\${NAME}\${VERSION}\${OS_ARCH}"
    
    if (-not (Test-Path "${BINARY}.exe")) {
        Build
    }
    
    New-Item -ItemType Directory -Path $targetDir -Force | Out-Null
    Move-Item -Path "${BINARY}.exe" -Destination $targetDir -Force -ErrorAction Stop
    Remove-Item -Path ".terraform.lock.hcl" -ErrorAction SilentlyContinue
    Write-Host "Installed in: $targetDir" -ForegroundColor Green
}

function TestAcc {
    $env:TF_ACC = "1"
    go test ./... -v -timeout 120m @args
}

function GenerateDoc {
    go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
}

function TestAccReferences {
    $env:TF_ACC = "1"
    go test -run "TestAccReference" (Get-GoPackages) -v -timeout 60m
}

function TestAccCompute {
    $env:TF_ACC = "1"
    go test -run "TestAccCompute" (Get-GoPackages) -v -timeout 60m
}

function TestAccKafka {
    $env:TF_ACC = "1"
    go test -run "TestAccKafka" (Get-GoPackages) -v -timeout 60m
}

# Обработка аргументов
if ($args.Count -eq 0) {
    Build
} else {
    switch ($args[0]) {
        "build" { Build }
        "release" { Release }
        "install_local" { InstallLocal }
        "testacc" { TestAcc @($args[1..($args.Count-1)]) }
        "generate_doc" { GenerateDoc }
        "testacc_references" { TestAccReferences }
        "testacc_compute" { TestAccCompute }
        "testacc_kafka" { TestAccKafka }
        default { Write-Host "Command not found: $($args[0])" -ForegroundColor Red }
    }
}