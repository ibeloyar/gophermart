package config

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

func TestRead_Defaults(t *testing.T) {
	resetFlags()
	os.Args = []string{"cmd"}

	for _, key := range []string{
		"RUN_ADDRESS", "DATABASE_URI", "ACCRUAL_SYSTEM_ADDRESS",
		"PASS_COST", "SECRET_KEY", "TOKEN_LIFETIME",
	} {
		os.Unsetenv(key)
	}

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
	resetFlags()
	os.Args = []string{"cmd",
		"-a=:3000",
		"-d=postgres://user:pass@localhost/db",
		"-r=http://accrual:8080",
		"-p=10",
		"-s=mysecret",
		"-h=1h",
	}

	os.Unsetenv("RUN_ADDRESS")
	os.Unsetenv("DATABASE_URI")

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
	resetFlags()
	os.Args = []string{"cmd"}

	os.Setenv("RUN_ADDRESS", ":9000")
	os.Setenv("DATABASE_URI", "env_db_url")
	os.Setenv("ACCRUAL_SYSTEM_ADDRESS", "http://env:9000")
	os.Setenv("PASS_COST", "12")
	os.Setenv("SECRET_KEY", "env_secret")
	os.Setenv("TOKEN_LIFETIME", "30m")
	defer func() {
		os.Unsetenv("RUN_ADDRESS")
		os.Unsetenv("DATABASE_URI")
		os.Unsetenv("ACCRUAL_SYSTEM_ADDRESS")
		os.Unsetenv("PASS_COST")
		os.Unsetenv("SECRET_KEY")
		os.Unsetenv("TOKEN_LIFETIME")
	}()

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
	resetFlags()
	os.Args = []string{"cmd", "-a=:8080"}

	os.Setenv("RUN_ADDRESS", ":9090")
	defer os.Unsetenv("RUN_ADDRESS")

	config, err := Read()
	require.NoError(t, err)

	require.Equal(t, ":9090", config.RunAddress)
}

func TestRead_EnvParseError(t *testing.T) {
	resetFlags()
	os.Args = []string{"cmd"}

	os.Setenv("TOKEN_LIFETIME", "invalid_duration")
	defer os.Unsetenv("TOKEN_LIFETIME")

	_, err := Read()
	require.Error(t, err)
}
