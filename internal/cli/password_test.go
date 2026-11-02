package cli

import (
	"os"
	"testing"
)

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
