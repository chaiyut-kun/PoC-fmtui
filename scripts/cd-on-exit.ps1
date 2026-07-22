function fm {
    # Check for TMPDIR environment variable, default to /tmp
    $tmpDir = if ($env:TMPDIR) { $env:TMPDIR } else { "/tmp" }
    
    # For Windows compatibility, fallback to TEMP if TMPDIR isn't set and /tmp doesn't exist
    if (-not $env:TMPDIR -and -not (Test-Path "/tmp")) {
        $tmpDir = $env:TEMP
    }
    
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
}
