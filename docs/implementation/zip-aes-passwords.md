# Feature: zip AES-256 passwords

## Goal

Create password-protected zip archives and extract/list them, without putting crypto policy inside the CLI command layer beyond resolving the password string.

## Technologies considered

| Option | Notes | Decision |
|--------|-------|----------|
| Stdlib `archive/zip` | No encryption API (rejected upstream) | Unencrypted writes only |
| ZipCrypto (traditional) | Weak; wide compatibility | Rejected as default |
| `github.com/yeka/zip` AES-256 | WinZip-compatible AES | **Chosen** |
| External `7z` | Heavy dependency | Rejected |

## Password resolution (CLI)

Order locked in SPEC:

1. `--password` (if flag present)
2. `ZIPSTAR_PASSWORD`
3. Interactive prompt on a TTY (stderr prompt; each character masked as `*`)

`-e` / `--encrypt` is required to create an encrypted archive so passwords are never applied accidentally. On a TTY, omitting `--password` / `ZIPSTAR_PASSWORD` opens a masked prompt (`*` per character).

## Implementation notes

Encrypted create path (`archiveEncrypted`):

```go
header.SetPassword(password)
header.SetEncryptionMethod(yezazip.AES256Encryption)
w, err := zw.CreateHeader(header)
```

Extract sets the password on each encrypted member before `Open()`. Wrong passwords surface as exit code `2` with an explicit message.

### Level limitation

`yeka/zip` registers a process-wide deflate compressor at level 5 and panics on re-registration. Therefore:

- Unencrypted: full `--level` via stdlib `Writer.RegisterCompressor`
- Encrypted: `--level 0` → store; `1`–`9` → library deflate

Documented in `docs/FORMATS.md`.

## Tests

`TestListAndEncrypt` — create with password, refuse wrong password, extract with correct password, assert `Encrypted` on list.
