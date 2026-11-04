# Bundled UnRAR binaries

zipstar prefers the official **UnRAR** CLI for RAR extract/list (speed + fidelity),
falling back to pure-Go `rardecode` when no binary is found.

## Layout

```
third_party/unrar/
  LICENSE.txt                 # UnRAR freeware license (required)
  darwin_arm64/unrar
  darwin_amd64/unrar
  linux_amd64/unrar
  windows_amd64/unrar.exe
```

Fetch locally:

```bash
./scripts/fetch-unrar.sh              # detect host OS/arch (uname)
./scripts/fetch-unrar.sh --all        # every supported release platform
./scripts/fetch-unrar.sh darwin_arm64 # explicit platform(s)
```

## Release assets

Ship alongside each zipstar binary (same directory), named:

| Asset name | Platform |
|------------|----------|
| `unrar-darwin-arm64` | macOS Apple Silicon |
| `unrar-darwin-amd64` | macOS Intel |
| `unrar-linux-amd64` | Linux x86_64 |
| `unrar-windows-amd64.exe` | Windows x86_64 |

At runtime zipstar looks for (in order):

1. `ZIPSTAR_UNRAR` env override  
2. `unrar` / `unrar.exe` / `unrar-<goos>-<goarch>[.exe]` next to the zipstar executable  
3. `third_party/unrar/<goos>_<goarch>/unrar[.exe]` (dev tree)  
4. `unrar` on `PATH`  
5. Pure-Go `rardecode` fallback  

Linux arm64 is not currently published by RARLab in the same package set; those hosts use `rardecode` unless the user provides `ZIPSTAR_UNRAR`.
