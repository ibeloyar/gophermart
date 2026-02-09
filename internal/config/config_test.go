package config

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func resetFlags(t *testing.T) {
	t.Helper()
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

func TestRead_Defaults(t *testing.T) {
	resetFlags(t)
	os.Args = []string{"cmd"}

	t.Setenv("RUN_ADDRESS", "")
	t.Setenv("DATABASE_URI", "")
	t.Setenv("ACCRUAL_SYSTEM_ADDRESS", "")
	t.Setenv("PASS_COST", "")
	t.Setenv("SECRET_KEY", "")
	t.Setenv("TOKEN_LIFETIME", "")

	config, err := Read()
	require.NoError(t, err)

	require.Equal(t, ":8080", config.RunAddress)
	require.Equal(t, "", config.DatabaseURI)
	require.Equal(t, "http://localhost:4000", config.AccrualSystemAddress)
	require.Equal(t, 3, config.PassCost)
	require.Equal(t, "secret", config.SecretKey)
	require.Equal(t, 3*time.Hour, config.TokenLifetime)
}

func TestRead_Flags(t *testing.T) {
	resetFlags(t)
	os.Args = []string{"cmd",
		"-a=:3000",
		"-d=postgres://user:pass@localhost/db",
		"-r=http://accrual:8080",
		"-p=10",
		"-s=mysecret",
		"-h=1h",
	}

	t.Setenv("RUN_ADDRESS", "")
	t.Setenv("DATABASE_URI", "")

	config, err := Read()
	require.NoError(t, err)

	require.Equal(t, ":3000", config.RunAddress)
	require.Equal(t, "postgres://user:pass@localhost/db", config.DatabaseURI)
	require.Equal(t, "http://accrual:8080", config.AccrualSystemAddress)
	require.Equal(t, 10, config.PassCost)
	require.Equal(t, "mysecret", config.SecretKey)
	require.Equal(t, time.Hour, config.TokenLifetime)
}

func TestRead_EnvVars(t *testing.T) {
	resetFlags(t)
	os.Args = []string{"cmd"}

	t.Setenv("RUN_ADDRESS", ":9000")
	t.Setenv("DATABASE_URI", "env_db_url")
	t.Setenv("ACCRUAL_SYSTEM_ADDRESS", "http://env:9000")
	t.Setenv("PASS_COST", "12")
	t.Setenv("SECRET_KEY", "env_secret")
	t.Setenv("TOKEN_LIFETIME", "30m")

	config, err := Read()
	require.NoError(t, err)

	require.Equal(t, ":9000", config.RunAddress)
	require.Equal(t, "env_db_url", config.DatabaseURI)
	require.Equal(t, "http://env:9000", config.AccrualSystemAddress)
	require.Equal(t, 12, config.PassCost)
	require.Equal(t, "env_secret", config.SecretKey)
	require.Equal(t, 30*time.Minute, config.TokenLifetime)
}

func TestRead_FlagsOverrideEnv(t *testing.T) {
	resetFlags(t)
	os.Args = []string{"cmd", "-a=:8080"}

	t.Setenv("RUN_ADDRESS", ":9090")

	config, err := Read()
	require.NoError(t, err)

	require.Equal(t, ":9090", config.RunAddress)
}

func TestRead_EnvParseError(t *testing.T) {
	resetFlags(t)
	os.Args = []string{"cmd"}

	t.Setenv("TOKEN_LIFETIME", "invalid_duration")

	_, err := Read()
	require.Error(t, err)
}
