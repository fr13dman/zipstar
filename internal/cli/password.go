package cli

import (
	"fmt"
	"io"
	"os"
	"unicode/utf8"

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

	state, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer func() { _ = term.Restore(fd, state) }()

	pw, err := readPasswordMasked(os.Stdin, os.Stderr)
	// Raw mode: emit CRLF so the next line of output starts cleanly.
	fmt.Fprint(os.Stderr, "\r\n")
	if err != nil {
		return "", err
	}
	return pw, nil
}

// readPasswordMasked reads a password from in, writing one '*' per rune to out.
// Supports UTF-8, backspace/delete, Enter to finish, and Ctrl-C to cancel.
func readPasswordMasked(in io.Reader, out io.Writer) (string, error) {
	var pw []rune
	var partial []byte
	buf := make([]byte, 1)

	for {
		_, err := in.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}

		b := buf[0]
		switch b {
		case '\r', '\n':
			return string(pw), nil
		case 0x03: // Ctrl-C
			return "", fmt.Errorf("interrupted")
		case 0x04: // Ctrl-D
			return string(pw), nil
		case 127, 8: // DEL / Backspace
			partial = nil
			if len(pw) == 0 {
				continue
			}
			pw = pw[:len(pw)-1]
			if _, err := io.WriteString(out, "\b \b"); err != nil {
				return "", err
			}
			continue
		}

		if b < 32 {
			// Ignore other control characters.
			continue
		}

		partial = append(partial, b)
		if !utf8.FullRune(partial) {
			continue
		}
		r, size := utf8.DecodeRune(partial)
		partial = partial[size:]
		if r == utf8.RuneError {
			continue
		}
		pw = append(pw, r)
		if _, err := io.WriteString(out, "*"); err != nil {
			return "", err
		}
	}
	return string(pw), nil
}
