package pgfs_test

import (
	"strings"
	"testing"

	pgfs "github.com/422UR4H/HxH_RPG_System/pkg"
)

func TestConfig_ConnString(t *testing.T) {
	c := pgfs.Config{
		DbName:        "testdb",
		DbUser:        "user",
		DbPass:        "pass",
		DbHost:        "localhost",
		DbPort:        "5432",
		DbSSLMode:     "disable",
		DbPoolMinSize: 2,
		DbPoolMaxSize: 10,
	}

	got := c.ConnString()
	want := "postgres://user:pass@localhost:5432/testdb?sslmode=disable&pool_min_conns=2&pool_max_conns=10"

	if got != want {
		t.Errorf("ConnString()\ngot  %q\nwant %q", got, want)
	}
}

func TestConfig_ConnString_EmptySSLMode_DefaultsToDisable(t *testing.T) {
	c := pgfs.Config{
		DbName: "testdb",
		DbUser: "user",
		DbPass: "pass",
		DbHost: "localhost",
		DbPort: "5432",
	}

	got := c.ConnString()
	if !strings.Contains(got, "sslmode=disable") {
		t.Errorf("expected sslmode=disable when SSLMode is empty, got %q", got)
	}
}

func TestConfig_ConnString_CustomSSLMode(t *testing.T) {
	c := pgfs.Config{
		DbName:    "prod",
		DbUser:    "admin",
		DbPass:    "secret",
		DbHost:    "db.example.com",
		DbPort:    "5433",
		DbSSLMode: "require",
	}

	got := c.ConnString()
	if !strings.Contains(got, "sslmode=require") {
		t.Errorf("expected sslmode=require, got %q", got)
	}
	if !strings.Contains(got, "db.example.com:5433") {
		t.Errorf("expected custom host:port, got %q", got)
	}
}
