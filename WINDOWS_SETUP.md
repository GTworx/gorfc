# Windows Setup Guide for gorfc

This guide explains how to build and use the `gorfc` SAP NetWeaver RFC library on Windows.

## Prerequisites

### 1. SAP NetWeaver RFC SDK
- **Location**: `C:\Users\gokha\Dropbox\_1GNDLF\SAP\nwrfcsdk\`
- **Required structure**:
  - `include/` - Header files
  - `lib/` - Library files (sapnwrfc.dll, sapucum.dll, etc.)
  - `bin/` - Runtime DLLs

✅ Already installed and configured in this project.

### 2. C Compiler (GCC)

The project requires a C compiler because it uses CGO to interface with the SAP RFC SDK.

#### Option A: TDM-GCC (Recommended)
1. Download from: https://jmeubank.github.io/tdm-gcc/download/
2. Install the **64-bit** version (TDM-GCC-64)
3. During installation:
   - ✅ Check "Add to PATH"
   - Use default installation directory: `C:\TDM-GCC-64\`
4. Restart your terminal after installation
5. Verify installation:
   ```bash
   gcc --version
   ```

#### Option B: MSYS2/MinGW-w64
1. Download MSYS2 from: https://www.msys2.org/
2. Install and open MSYS2 terminal
3. Run:
   ```bash
   pacman -Syu
   pacman -S mingw-w64-x86_64-gcc
   ```
4. Add `C:\msys64\mingw64\bin` to your system PATH

#### Option C: Visual Studio Build Tools
1. Download from: https://visualstudio.microsoft.com/downloads/
2. Install "Desktop development with C++" workload
3. Use "x64 Native Tools Command Prompt" for building

### 3. Go Environment Configuration

Set the CGO environment variable:

#### PowerShell
```powershell
# Temporary (current session only)
$env:CGO_ENABLED=1

# Permanent (current user)
[System.Environment]::SetEnvironmentVariable('CGO_ENABLED', '1', 'User')

# Permanent (system-wide, requires admin)
[System.Environment]::SetEnvironmentVariable('CGO_ENABLED', '1', 'Machine')
```

#### CMD
```cmd
# Temporary (current session only)
set CGO_ENABLED=1

# Permanent (current user)
setx CGO_ENABLED 1
```

#### Bash/Git Bash/MSYS2
```bash
# Temporary (current session only)
export CGO_ENABLED=1

# Permanent (add to ~/.bashrc or ~/.bash_profile)
echo 'export CGO_ENABLED=1' >> ~/.bashrc
```

## Project Configuration

### Files Modified

1. **gorfc/gorfc.go** - Updated build constraints and SDK paths:
   - Build tags: `//go:build (linux && cgo) || (darwin && cgo) || (windows && cgo)`
   - Include path: `-IC:/Users/gokha/Dropbox/_1GNDLF/SAP/nwrfcsdk/include/`
   - Library path: `-LC:/Users/gokha/Dropbox/_1GNDLF/SAP/nwrfcsdk/lib/`

2. **.vscode/settings.json** - VS Code configuration for gopls:
   ```json
   {
       "go.toolsEnvVars": {
           "CGO_ENABLED": "1"
       },
       "gopls": {
           "build.env": {
               "CGO_ENABLED": "1"
           }
       }
   }
   ```

## Building the Project

### PowerShell (Windows)

### 1. Build the library
```powershell
cd c:/Users/gokha/Dropbox/_1GNDLF/gorfc
$env:CGO_ENABLED=1
go build -v ./gorfc
```

### 2. Build your application
```powershell
$env:CGO_ENABLED=1
go build -v sap_rfc_reader.go
```

### Bash/Git Bash/MSYS2

### 1. Build the library
```bash
cd c:/Users/gokha/Dropbox/_1GNDLF/gorfc
CGO_ENABLED=1 go build -v ./gorfc
```

### 2. Build your application
```bash
CGO_ENABLED=1 go build -v sap_rfc_reader.go
```

### 3. Run your application
Before running, ensure SAP RFC SDK DLLs are accessible:

#### PowerShell
```powershell
# Option A: Add to PATH temporarily
$env:PATH += ";C:\Users\gokha\Dropbox\_1GNDLF\SAP\nwrfcsdk\lib"

# Option B: Copy DLLs to your application directory (Recommended)
Copy-Item "C:\Users\gokha\Dropbox\_1GNDLF\SAP\nwrfcsdk\lib\*.dll" .

# Then run
.\sap_rfc_reader.exe
```

#### Bash/Git Bash/MSYS2
```bash
# Option A: Add to PATH temporarily
export PATH="$PATH:/c/Users/gokha/Dropbox/_1GNDLF/SAP/nwrfcsdk/lib"

# Option B: Copy DLLs to your application directory (Recommended)
cp C:/Users/gokha/Dropbox/_1GNDLF/SAP/nwrfcsdk/lib/*.dll .

# Then run
./sap_rfc_reader.exe
```

## Troubleshooting

### Error: "build constraints exclude all Go files"
- **Cause**: CGO is disabled
- **Solution**: Set `CGO_ENABLED=1` and ensure GCC is installed

### Error: 'gcc' not found
- **Cause**: C compiler not installed or not in PATH
- **Solution**: Install GCC (see Prerequisites above) and verify with `gcc --version`

### Error: Cannot find sapnwrfc.h
- **Cause**: SAP SDK path incorrect
- **Solution**: Verify SDK is at `C:\Users\gokha\Dropbox\_1GNDLF\SAP\nwrfcsdk\` and has `include/` directory

### Error: Cannot find -lsapnwrfc
- **Cause**: SAP SDK libraries not found
- **Solution**: Verify SDK has `lib/` directory with `sapnwrfc.lib` and `sapucum.lib`

### Runtime Error: DLL not found
- **Cause**: SAP RFC SDK DLLs not in PATH
- **Solution**: Add SDK `lib/` directory to PATH or copy DLLs to application directory

### VS Code showing "No packages found" warning
- **Cause**: gopls not configured for CGO
- **Solution**: Reload VS Code window (Ctrl+Shift+P → "Reload Window")

## Environment Summary

### PowerShell
```powershell
# Required environment variables
$env:CGO_ENABLED=1
$env:PATH += ";C:\TDM-GCC-64\bin;C:\Users\gokha\Dropbox\_1GNDLF\SAP\nwrfcsdk\lib"

# Verify setup
go env CGO_ENABLED    # Should show: 1
gcc --version         # Should show GCC version
go version           # Should show Go version
```

### Bash/Git Bash/MSYS2
```bash
# Required environment variables
export CGO_ENABLED=1
export PATH="$PATH:/c/TDM-GCC-64/bin:/c/Users/gokha/Dropbox/_1GNDLF/SAP/nwrfcsdk/lib"

# Verify setup
go env CGO_ENABLED    # Should show: 1
gcc --version         # Should show GCC version
go version           # Should show Go version
```

## Next Steps

After successful setup:

1. Test the connection to your SAP system
2. Verify RFC_READ_TABLE function works
3. Check SAP credentials and network connectivity
4. Review SAP connection parameters in `sap_rfc_reader.go`

## Additional Resources

- gorfc GitHub: https://github.com/SAP/gorfc
- SAP NW RFC SDK Documentation: SAP Support Portal
- CGO Documentation: https://pkg.go.dev/cmd/cgo

## Notes

- This project uses CGO, so cross-compilation is more complex
- Build times will be longer due to C compilation
- The SAP RFC SDK must match your system architecture (64-bit)
- Some linker flags (like `-pie`, `-fPIE`, `-OPT:REF`, `-LTCG`) may need adjustment depending on your GCC version
