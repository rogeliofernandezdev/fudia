$ErrorActionPreference = 'Stop'
$taskRepository = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
$previousDatabaseURL = $env:DATABASE_URL
$previousProcessorTestDatabaseURL = $env:FUDIA_IMAGE_PROCESSOR_TEST_DATABASE_URL
$testContainerID = $null
try {
    $testContainerName = 'fudia-images-test-' + [Guid]::NewGuid().ToString('N').Substring(0,12)
    $testContainerID = (docker run --detach --rm --name $testContainerName -e POSTGRES_USER=foods -e POSTGRES_PASSWORD=foods -e POSTGRES_DB=foods_test -p 127.0.0.1::5432 postgres:16).Trim()
    if ($LASTEXITCODE -ne 0 -or $testContainerID -notmatch '^[a-f0-9]{64}$') { throw 'No se pudo crear PostgreSQL temporal.' }
    for ($attempt=0; $attempt -lt 30; $attempt++) {
        docker exec $testContainerID pg_isready -U foods -d foods_test *> $null
        if ($LASTEXITCODE -eq 0) { break }
        Start-Sleep -Milliseconds 500
    }
    $testSocket = (docker port $testContainerID 5432/tcp).Trim()
    if ($testSocket -notmatch '^127\.0\.0\.1:\d+$') { throw 'Socket temporal inesperado.' }
    $env:DATABASE_URL = 'postgres://foods:foods@' + $testSocket + '/foods_test?sslmode=disable'
    $env:FUDIA_IMAGE_PROCESSOR_TEST_DATABASE_URL = $env:DATABASE_URL
    Push-Location (Join-Path $taskRepository 'foods-backend')
    try {
        go run ./cmd/migrate
        if ($LASTEXITCODE -ne 0) { throw 'Migraciones de prueba fallaron.' }
        go test -p 2 ./...
        if ($LASTEXITCODE -ne 0) { throw 'Pruebas del backend fallaron.' }
        go vet ./...
        if ($LASTEXITCODE -ne 0) { throw 'Vet falló.' }
        go build ./cmd/api ./cmd/migrate
        if ($LASTEXITCODE -ne 0) { throw 'Build falló.' }
    } finally { Pop-Location }
    Push-Location (Join-Path $taskRepository 'fudia-image-processor')
    try {
        go test -p 2 ./...
        if ($LASTEXITCODE -ne 0) { throw 'Pruebas del procesador fallaron.' }
        go vet ./...
        if ($LASTEXITCODE -ne 0) { throw 'Vet del procesador falló.' }
        go build ./cmd/processor
        if ($LASTEXITCODE -ne 0) { throw 'Build del procesador falló.' }
    } finally { Pop-Location }
} finally {
    if ($testContainerID -match '^[a-f0-9]{64}$') { docker stop $testContainerID }
    if ($null -eq $previousDatabaseURL) { Remove-Item Env:DATABASE_URL -ErrorAction SilentlyContinue }
    else { $env:DATABASE_URL = $previousDatabaseURL }
    if ($null -eq $previousProcessorTestDatabaseURL) { Remove-Item Env:FUDIA_IMAGE_PROCESSOR_TEST_DATABASE_URL -ErrorAction SilentlyContinue }
    else { $env:FUDIA_IMAGE_PROCESSOR_TEST_DATABASE_URL = $previousProcessorTestDatabaseURL }
}
