# internal/formats/zip

Zip backend for zipstar.

- **Write (plain):** `archive/zip` with per-Writer flate `--level`
- **Write (AES):** `github.com/yeka/zip` with AES-256
- **Read / list:** `yeka/zip` (handles both plain and encrypted)

See `docs/implementation/zip-aes-passwords.md` for why both libraries appear.
