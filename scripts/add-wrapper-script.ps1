Add-Content -Path $PROFILE -Value 'function fm {
    # [System.IO.Path]::GetTempPath() accurately matches Go's os.TempDir() on both Windows and Linux/macOS
    $tmpDir = [System.IO.Path]::GetTempPath()
    $tmpFile = Join-Path $tmpDir "fmtui-dir"

    # Execute the actual fm binary (not this function)
    $cmd = Get-Command fm -CommandType Application -ErrorAction SilentlyContinue
    if (-not $cmd) {
        Write-Error "fm command not found in PATH."
        return
    }
    
    & $cmd @args

    if (Test-Path $tmpFile) {
        $dir = Get-Content $tmpFile -Raw
        $dir = $dir.Trim()
        Remove-Item $tmpFile -Force
        if (Test-Path -Path $dir -PathType Container) {
            Set-Location $dir
        }
    }
}'
