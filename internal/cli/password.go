package cli

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

const passwordEnv = "ZIPSTAR_PASSWORD"

// PasswordSources holds the CLI inputs used to resolve a password.
// Precedence: Flag → ZIPSTAR_PASSWORD → interactive prompt.
type PasswordSources struct {
	Flag    string // --password value (may be empty if unset)
	FlagSet bool   // true when --password was explicitly passed
	Prompt  bool   // when true and nothing else supplies a password, prompt on a TTY
}

// ResolvePassword returns the password using the locked precedence order.
// If needPassword is false, returns empty string with no error.
func ResolvePassword(needPassword bool, src PasswordSources) (string, error) {
	if !needPassword {
		return "", nil
	}

	if src.FlagSet {
		if src.Flag == "" {
			return "", Usagef("password is empty: pass a non-empty --password, set %s, or omit --password to be prompted", passwordEnv)
		}
		return src.Flag, nil
	}

	if env := os.Getenv(passwordEnv); env != "" {
		return env, nil
	}

	if !src.Prompt {
		return "", Usagef("password required: pass --password, set %s, or run in a terminal to be prompted", passwordEnv)
	}

	pw, err := promptPassword("Password: ")
	if err != nil {
		return "", Runtimef("could not read password: %v", err)
	}
	if pw == "" {
		return "", Usagef("password is empty: provide a non-empty password")
	}
	return pw, nil
}

func promptPassword(prompt string) (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", fmt.Errorf("stdin is not a terminal (set --password or %s for non-interactive use)", passwordEnv)
	}
	fmt.Fprint(os.Stderr, prompt)
	raw, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(raw), "\r\n"), nil
}
