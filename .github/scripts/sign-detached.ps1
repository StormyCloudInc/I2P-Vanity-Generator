<#
  sign-detached.ps1
  Create a PKCS#7 detached signature (.p7s) for any file using a certificate
  from the Windows certificate store (e.g. SimplySign virtual smart card).

  The .p7s file can be verified on any platform with:
    openssl cms -verify -in file.p7s -content file -inform DER -noverify
#>

param(
    [Parameter(Mandatory=$true)]
    [string]$FilePath,

    [string]$CertificateSHA1 = $env:CERTUM_CERTIFICATE_SHA1
)

if (-not $CertificateSHA1) {
    Write-Host "ERROR: CERTUM_CERTIFICATE_SHA1 not set"
    exit 1
}

if (-not (Test-Path $FilePath)) {
    Write-Host "ERROR: File not found: $FilePath"
    exit 1
}

Write-Host "=== Creating Detached Signature ==="
Write-Host "File: $FilePath"
Write-Host "Certificate SHA1: $($CertificateSHA1.Substring(0, [Math]::Min(16, $CertificateSHA1.Length)))..."

# Find the certificate in the store
$store = New-Object System.Security.Cryptography.X509Certificates.X509Store("My", "CurrentUser")
$store.Open("ReadOnly")
$cert = $store.Certificates | Where-Object { $_.Thumbprint -eq $CertificateSHA1 }
$store.Close()

if (-not $cert) {
    Write-Host "ERROR: Certificate not found in CurrentUser\My store"
    Write-Host "Available certificates:"
    $store.Open("ReadOnly")
    foreach ($c in $store.Certificates) {
        Write-Host "  - $($c.Thumbprint) | $($c.Subject)"
    }
    $store.Close()
    exit 1
}

Write-Host "Found certificate: $($cert.Subject)"

# Read the file content
$content = [System.IO.File]::ReadAllBytes((Resolve-Path $FilePath))
Write-Host "File size: $($content.Length) bytes"

# Create detached CMS/PKCS#7 signature
Add-Type -AssemblyName System.Security
$contentInfo = New-Object System.Security.Cryptography.Pkcs.ContentInfo(,$content)
$signedCms = New-Object System.Security.Cryptography.Pkcs.SignedCms($contentInfo, $true)
$signer = New-Object System.Security.Cryptography.Pkcs.CmsSigner($cert)
$signer.DigestAlgorithm = New-Object System.Security.Cryptography.Oid("2.16.840.1.101.3.4.2.1")  # SHA-256
$signer.IncludeOption = [System.Security.Cryptography.X509Certificates.X509IncludeOption]::EndCertOnly

try {
    $signedCms.ComputeSignature($signer)
} catch {
    Write-Host "ERROR: Failed to compute signature - $_"
    exit 1
}

$outputPath = "$FilePath.p7s"
[System.IO.File]::WriteAllBytes($outputPath, $signedCms.Encode())

Write-Host "Detached signature written to: $outputPath"
Write-Host "Signature size: $((Get-Item $outputPath).Length) bytes"
Write-Host ""
Write-Host "=== Detached signature complete ==="
