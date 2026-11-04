package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestReadPasswordMasked(t *testing.T) {
	in := strings.NewReader("s3cret\n")
	var out bytes.Buffer
	got, err := readPasswordMasked(in, &out)
	if err != nil {
		t.Fatal(err)
	}
	if got != "s3cret" {
		t.Fatalf("got password %q", got)
	}
	if out.String() != "******" {
		t.Fatalf("mask = %q, want ******", out.String())
	}
}

func TestReadPasswordMaskedBackspace(t *testing.T) {
	// type "ab", backspace, "c", enter
	in := bytes.NewReader([]byte{'a', 'b', 127, 'c', '\n'})
	var out bytes.Buffer
	got, err := readPasswordMasked(in, &out)
	if err != nil {
		t.Fatal(err)
	}
	if got != "ac" {
		t.Fatalf("got password %q", got)
	}
	if out.String() != "**\b \b*" {
		t.Fatalf("unexpected mask stream %q", out.String())
	}
}

func TestReadPasswordMaskedCtrlC(t *testing.T) {
	in := bytes.NewReader([]byte{'x', 0x03})
	var out bytes.Buffer
	_, err := readPasswordMasked(in, &out)
	if err == nil {
		t.Fatal("expected interrupt error")
	}
}

func TestReadPasswordMaskedUTF8(t *testing.T) {
	in := strings.NewReader("päss\n")
	var out bytes.Buffer
	got, err := readPasswordMasked(in, &out)
	if err != nil {
		t.Fatal(err)
	}
	if got != "päss" {
		t.Fatalf("got %q", got)
	}
	if out.String() != "****" {
		t.Fatalf("mask = %q", out.String())
	}
}

func TestReadPasswordMaskedEOF(t *testing.T) {
	in := strings.NewReader("ab")
	var out bytes.Buffer
	got, err := readPasswordMasked(in, &out)
	if err != nil && err != io.EOF {
		t.Fatal(err)
	}
	if got != "ab" {
		t.Fatalf("got %q", got)
	}
}

func TestResolvePasswordPrecedence(t *testing.T) {
	t.Setenv(passwordEnv, "from-env")

	pw, err := ResolvePassword(true, PasswordSources{Flag: "from-flag", FlagSet: true})
	if err != nil {
		t.Fatal(err)
	}
	if pw != "from-flag" {
		t.Fatalf("flag should win, got %q", pw)
	}

	pw, err = ResolvePassword(true, PasswordSources{})
	if err != nil {
		t.Fatal(err)
	}
	if pw != "from-env" {
		t.Fatalf("env should win when no flag, got %q", pw)
	}
}

func TestResolvePasswordNotNeeded(t *testing.T) {
	pw, err := ResolvePassword(false, PasswordSources{FlagSet: true, Flag: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if pw != "" {
		t.Fatalf("expected empty, got %q", pw)
	}
}

func TestResolvePasswordEmptyFlag(t *testing.T) {
	_, err := ResolvePassword(true, PasswordSources{FlagSet: true, Flag: ""})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolvePasswordMissing(t *testing.T) {
	os.Unsetenv(passwordEnv)
	_, err := ResolvePassword(true, PasswordSources{Prompt: false})
	if err == nil {
		t.Fatal("expected error")
	}
}
