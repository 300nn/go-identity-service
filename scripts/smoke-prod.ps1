param(
    [string]$EnvFile = ".env.prod",
    [string]$ComposeFile = "docker-compose.prod.yml",
    [int]$TimeoutSeconds = 90
)

$ErrorActionPreference = "Stop"

function Write-Step
{
    param([string]$Message)
    Write-Host ""
    Write-Host "==> $Message"
}

function Invoke-HttpCheck
{
    param(
        [string]$Url,
        [int]$ExpectedStatus = 200
    )

    try
    {
        $response = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 5
        if ($response.StatusCode -ne $ExpectedStatus)
        {
            throw "Expected HTTP $ExpectedStatus from $Url, got $( $response.StatusCode )"
        }

        Write-Host "OK $Url -> $( $response.StatusCode )"
    }
    catch
    {
        throw "HTTP check failed for $Url. $( $_.Exception.Message )"
    }
}

if (-not (Test-Path $EnvFile))
{
    throw "Env file '$EnvFile' does not exist. Copy .env.prod.example to .env.prod first."
}

if (-not (Test-Path $ComposeFile))
{
    throw "Compose file '$ComposeFile' does not exist."
}

Write-Step "Pull images"
docker compose -f $ComposeFile --env-file $EnvFile pull

Write-Step "Start production-like stack"
docker compose -f $ComposeFile --env-file $EnvFile up -d

Write-Step "Show containers"
docker compose -f $ComposeFile --env-file $EnvFile ps

Write-Step "Wait for readiness"

$deadline = (Get-Date).AddSeconds($TimeoutSeconds)
$ready = $false

while ((Get-Date) -lt $deadline)
{
    try
    {
        $response = Invoke-WebRequest -Uri "http://localhost:8080/ready" -UseBasicParsing -TimeoutSec 3
        if ($response.StatusCode -eq 200)
        {
            $ready = $true
            break
        }
    }
    catch
    {
        Start-Sleep -Seconds 2
    }

    Start-Sleep -Seconds 2
}

if (-not $ready)
{
    Write-Host ""
    Write-Host "Readiness check failed. API logs:"
    docker compose -f $ComposeFile --env-file $EnvFile logs --tail=100 api

    Write-Host ""
    Write-Host "Migrate logs:"
    docker compose -f $ComposeFile --env-file $EnvFile logs --tail=100 migrate

    throw "Service did not become ready within $TimeoutSeconds seconds."
}

Write-Step "Check HTTP endpoints"
Invoke-HttpCheck "http://localhost:8080/health"
Invoke-HttpCheck "http://localhost:8080/ready"
Invoke-HttpCheck "http://localhost:8080/metrics"

Write-Step "Check database tables"

docker exec identity-service-postgres `
    psql -U identity_service -d identity_service `
    -c "select version_id, is_applied from goose_db_version order by id;"

docker exec identity-service-postgres `
    psql -U identity_service -d identity_service `
    -c "select to_regclass('public.outbox_events') as outbox_events;"

Write-Step "Smoke test passed"