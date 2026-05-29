param(
    [string]$Config = (Join-Path $PSScriptRoot "account-backup-sync.env"),
    [switch]$DryRun
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Read-DotEnv {
    param([string]$Path)

    $values = @{}
    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Config file not found: $Path"
    }

    foreach ($line in Get-Content -LiteralPath $Path) {
        $trimmed = $line.Trim()
        if ($trimmed -eq "" -or $trimmed.StartsWith("#")) {
            continue
        }
        $parts = $trimmed -split "=", 2
        if ($parts.Count -ne 2) {
            continue
        }
        $key = $parts[0].Trim()
        $value = $parts[1].Trim()
        if (($value.StartsWith('"') -and $value.EndsWith('"')) -or ($value.StartsWith("'") -and $value.EndsWith("'"))) {
            $value = $value.Substring(1, $value.Length - 2)
        }
        $values[$key] = $value
    }
    return $values
}

function Get-RequiredConfig {
    param(
        [hashtable]$Values,
        [string]$Name
    )
    if (-not $Values.ContainsKey($Name) -or [string]::IsNullOrWhiteSpace($Values[$Name])) {
        throw "Missing required config: $Name"
    }
    return $Values[$Name].Trim()
}

function Get-OptionalConfig {
    param(
        [hashtable]$Values,
        [string]$Name,
        [string]$DefaultValue
    )
    if (-not $Values.ContainsKey($Name) -or [string]::IsNullOrWhiteSpace($Values[$Name])) {
        return $DefaultValue
    }
    return $Values[$Name].Trim()
}

function New-AuthHeaders {
    param([hashtable]$Values)

    $headers = @{}
    if ($Values.ContainsKey("SUB2API_ADMIN_API_KEY") -and -not [string]::IsNullOrWhiteSpace($Values["SUB2API_ADMIN_API_KEY"])) {
        $headers["x-api-key"] = $Values["SUB2API_ADMIN_API_KEY"].Trim()
        return $headers
    }
    if ($Values.ContainsKey("SUB2API_BEARER_TOKEN") -and -not [string]::IsNullOrWhiteSpace($Values["SUB2API_BEARER_TOKEN"])) {
        $headers["Authorization"] = "Bearer " + $Values["SUB2API_BEARER_TOKEN"].Trim()
        return $headers
    }
    throw "Set SUB2API_ADMIN_API_KEY or SUB2API_BEARER_TOKEN"
}

function Join-Url {
    param(
        [string]$BaseUrl,
        [string]$Path
    )
    return $BaseUrl.TrimEnd("/") + "/" + $Path.TrimStart("/")
}

function Import-RsaPrivateKeyFromPem {
    param([string]$PrivateKeyFile)

    $pem = Get-Content -Raw -LiteralPath $PrivateKeyFile
    $rsa = [System.Security.Cryptography.RSA]::Create()
    try {
        $rsa.ImportFromPem($pem)
    }
    catch {
        throw "Failed to import private key. Please run this script with PowerShell 7+ and use a PEM PKCS#8/RSA private key. $($_.Exception.Message)"
    }
    return $rsa
}

function ConvertFrom-EncryptedSnapshot {
    param(
        [string]$EncryptedPath,
        [string]$JsonPath,
        [System.Security.Cryptography.RSA]$PrivateKey
    )

    $envelope = Get-Content -Raw -LiteralPath $EncryptedPath | ConvertFrom-Json
    $encryptedKey = [Convert]::FromBase64String($envelope.encrypted_key)
    $nonce = [Convert]::FromBase64String($envelope.nonce)
    $ciphertext = [Convert]::FromBase64String($envelope.ciphertext)
    $tag = [Convert]::FromBase64String($envelope.tag)

    $aesKey = $PrivateKey.Decrypt($encryptedKey, [System.Security.Cryptography.RSAEncryptionPadding]::OaepSHA256)
    $plaintext = [byte[]]::new($ciphertext.Length)
    $associatedData = [byte[]]::new(0)
    $gcm = [System.Security.Cryptography.AesGcm]::new($aesKey)
    try {
        $gcm.Decrypt($nonce, $ciphertext, $tag, $plaintext, $associatedData)
    }
    finally {
        $gcm.Dispose()
    }
    [System.IO.File]::WriteAllBytes($JsonPath, $plaintext)
}

function Remove-OldBackups {
    param(
        [string]$Directory,
        [string]$Filter,
        [int]$RetainCount
    )

    if ($RetainCount -le 0 -or -not (Test-Path -LiteralPath $Directory)) {
        return
    }
    $files = Get-ChildItem -LiteralPath $Directory -Filter $Filter -File | Sort-Object LastWriteTimeUtc -Descending
    $remove = @($files | Select-Object -Skip $RetainCount)
    foreach ($file in $remove) {
        Remove-Item -LiteralPath $file.FullName -Force
        Write-Host "Removed old backup: $($file.FullName)"
    }
}

$cfg = Read-DotEnv -Path $Config
$baseUrl = Get-RequiredConfig -Values $cfg -Name "SUB2API_BASE_URL"
$privateKeyFile = Get-RequiredConfig -Values $cfg -Name "ACCOUNT_BACKUP_PRIVATE_KEY_FILE"
$outputDir = Get-OptionalConfig -Values $cfg -Name "ACCOUNT_BACKUP_OUTPUT_DIR" -DefaultValue (Join-Path $PSScriptRoot "..\account_backups")
$encryptedDir = Get-OptionalConfig -Values $cfg -Name "ACCOUNT_BACKUP_ENCRYPTED_DIR" -DefaultValue (Join-Path $outputDir "encrypted")
$retainCount = [int](Get-OptionalConfig -Values $cfg -Name "ACCOUNT_BACKUP_RETAIN_COUNT" -DefaultValue "10")
$listUrl = Join-Url -BaseUrl $baseUrl -Path "/api/v1/admin/accounts/backups"

if ($DryRun) {
    Write-Host "Dry run OK"
    Write-Host "Base URL: $baseUrl"
    Write-Host "Output dir: $outputDir"
    Write-Host "Encrypted cache dir: $encryptedDir"
    Write-Host "Retain count: $retainCount"
    Write-Host "List URL: $listUrl"
    return
}

if (-not (Test-Path -LiteralPath $privateKeyFile)) {
    throw "Private key file not found: $privateKeyFile"
}
New-Item -ItemType Directory -Force -Path $outputDir | Out-Null
New-Item -ItemType Directory -Force -Path $encryptedDir | Out-Null

$headers = New-AuthHeaders -Values $cfg
$privateKey = Import-RsaPrivateKeyFromPem -PrivateKeyFile $privateKeyFile

try {
    $listResp = Invoke-RestMethod -Method Get -Uri $listUrl -Headers $headers
    $items = @($listResp.data.items)
    foreach ($item in $items) {
        $name = [string]$item.name
        if ([string]::IsNullOrWhiteSpace($name) -or -not $name.EndsWith(".json.enc")) {
            continue
        }

        $encryptedPath = Join-Path $encryptedDir $name
        $jsonName = $name.Substring(0, $name.Length - ".enc".Length)
        $jsonPath = Join-Path $outputDir $jsonName

        if (-not (Test-Path -LiteralPath $encryptedPath)) {
            $downloadUrl = Join-Url -BaseUrl $baseUrl -Path ("/api/v1/admin/accounts/backups/" + [System.Uri]::EscapeDataString($name))
            Invoke-WebRequest -Method Get -Uri $downloadUrl -Headers $headers -OutFile $encryptedPath | Out-Null
            Write-Host "Downloaded encrypted snapshot: $name"
        }

        if (-not (Test-Path -LiteralPath $jsonPath)) {
            ConvertFrom-EncryptedSnapshot -EncryptedPath $encryptedPath -JsonPath $jsonPath -PrivateKey $privateKey
            Write-Host "Decrypted snapshot: $jsonPath"
        }
    }

    Remove-OldBackups -Directory $outputDir -Filter "account-data-*.json" -RetainCount $retainCount
    Remove-OldBackups -Directory $encryptedDir -Filter "account-data-*.json.enc" -RetainCount $retainCount
}
finally {
    $privateKey.Dispose()
}
