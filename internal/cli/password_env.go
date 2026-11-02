package cli

import "os"

func envPassword() string {
	return os.Getenv(passwordEnv)
}
