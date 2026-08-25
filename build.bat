@echo off
setlocal
cd /d "%~dp0"

where go >nul 2>nul || (
  echo Go is not installed or not in PATH.
  exit /b 1
)

echo [1/4] Preparing modules...
go mod tidy || exit /b 1

echo [2/4] Installing resource compiler...
go install github.com/akavel/rsrc@latest || exit /b 1

echo [3/4] Embedding manifest and icon...
"%USERPROFILE%\go\bin\rsrc.exe" -manifest app.manifest -ico assets\app.ico -o rsrc.syso || exit /b 1

echo [4/4] Building SimpleShare.exe...
go build -trimpath -ldflags="-s -w -H windowsgui" -o SimpleShare.exe . || exit /b 1

echo.
echo Build complete: SimpleShare.exe
endlocal
