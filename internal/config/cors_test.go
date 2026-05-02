package config_test

import (
	"os"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/config"
)

func TestLoadCORS_DefaultOrigins(t *testing.T) {
	if err := os.Unsetenv("ALLOWED_ORIGINS"); err != nil {
		t.Fatal(err)
	}

	c := config.LoadCORS()

	if len(c.AllowedOrigins) != 2 {
		t.Fatalf("got %d origins, want 2", len(c.AllowedOrigins))
	}
	if c.AllowedOrigins[0] != "http://localhost:5173" {
		t.Errorf("got origin[0] %q, want %q", c.AllowedOrigins[0], "http://localhost:5173")
	}
	if c.AllowedOrigins[1] != "http://127.0.0.1:5173" {
		t.Errorf("got origin[1] %q, want %q", c.AllowedOrigins[1], "http://127.0.0.1:5173")
	}
}

func TestLoadCORS_CustomOrigins(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com, https://api.example.com")

	c := config.LoadCORS()

	if len(c.AllowedOrigins) != 2 {
		t.Fatalf("got %d origins, want 2", len(c.AllowedOrigins))
	}
	if c.AllowedOrigins[0] != "https://app.example.com" {
		t.Errorf("got origin[0] %q, want %q", c.AllowedOrigins[0], "https://app.example.com")
	}
	if c.AllowedOrigins[1] != "https://api.example.com" {
		t.Errorf("got origin[1] %q, want %q", c.AllowedOrigins[1], "https://api.example.com")
	}
}

func TestLoadCORS_FixedFields(t *testing.T) {
	c := config.LoadCORS()

	if !c.AllowCredentials {
		t.Error("expected AllowCredentials to be true")
	}
	if c.MaxAge != 300 {
		t.Errorf("got MaxAge %d, want 300", c.MaxAge)
	}
	if len(c.AllowedMethods) != 5 {
		t.Errorf("got %d methods, want 5", len(c.AllowedMethods))
	}
}
