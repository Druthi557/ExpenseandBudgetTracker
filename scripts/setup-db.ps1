# Creates the application login role and databases named in the repo-root .env
# (DATABASE_URL / TEST_DATABASE_URL), using the role password already stored there.
# Prompts once for the postgres superuser password; never echoes or saves it.
#
# Usage:  powershell -ExecutionPolicy Bypass -File scripts\setup-db.ps1

$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$envFile = Join-Path $root '.env'
if (-not (Test-Path $envFile)) {
    throw ".env not found at $envFile - copy .env.example to .env and fill it in first."
}

function Get-UrlValue([string]$line, [string]$field) {
    # postgres://user:password@host:port/dbname -> password or dbname
    $url = $line.Substring($line.IndexOf('=') + 1).Trim()
    if ($field -eq 'password') {
        $afterScheme = $url.Substring($url.IndexOf('://') + 3)
        $creds = $afterScheme.Substring(0, $afterScheme.IndexOf('@'))
        return $creds.Substring($creds.IndexOf(':') + 1)
    }
    return $url.Substring($url.LastIndexOf('/') + 1)
}

$appDb = $null; $testDb = $null; $pebtPassword = $null
foreach ($line in Get-Content $envFile) {
    if ($line -like 'DATABASE_URL=*')        { $appDb = Get-UrlValue $line 'dbname' }
    if ($line -like 'TEST_DATABASE_URL=*')   { $testDb = Get-UrlValue $line 'dbname'; $pebtPassword = Get-UrlValue $line 'password' }
}
if (-not $appDb -or -not $testDb -or -not $pebtPassword) {
    throw 'DATABASE_URL / TEST_DATABASE_URL with user:password@host/db not found in .env'
}

$psql = "$env:ProgramFiles\PostgreSQL\18\bin\psql.exe"
if (-not (Test-Path $psql)) { throw "psql not found at $psql - adjust the path in this script." }

$sec = Read-Host 'postgres superuser password' -AsSecureString
$env:PGPASSWORD = [Runtime.InteropServices.Marshal]::PtrToStringAuto(
    [Runtime.InteropServices.Marshal]::SecureStringToBSTR($sec))

# Create or reset the pebt role (same password as in .env). The role SQL assumes
# the password contains no single quotes (regenerate it if it does).
$roleSql = @"
DO `$$ BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'pebt') THEN
    CREATE ROLE pebt LOGIN PASSWORD '$pebtPassword';
  ELSE
    ALTER ROLE pebt WITH LOGIN PASSWORD '$pebtPassword';
  END IF;
END `$$;
"@
& $psql -U postgres -v ON_ERROR_STOP=1 -c $roleSql
if ($LASTEXITCODE -ne 0) { throw "role setup failed (psql exit $LASTEXITCODE)" }

foreach ($db in @($appDb, $testDb)) {
    $exists = & $psql -U postgres -t -A -c "SELECT 1 FROM pg_database WHERE datname = '$db'"
    if ("$exists".Trim() -eq '1') {
        Write-Host "database '$db' already exists - leaving it as is."
    } else {
        & $psql -U postgres -v ON_ERROR_STOP=1 -c "CREATE DATABASE $db OWNER pebt"
        if ($LASTEXITCODE -ne 0) { throw "creating '$db' failed (psql exit $LASTEXITCODE)" }
        Write-Host "created database '$db'."
    }
}

Remove-Item Env:PGPASSWORD
Write-Host "Done. Next: cd backend; go run ./cmd/migrate up  (then run the test suite with TEST_DATABASE_URL set)."
