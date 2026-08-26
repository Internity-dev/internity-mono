package redisx

import "testing"

func TestResolveOptions(t *testing.T) {
	t.Run("bare docker-compose service name and port", func(t *testing.T) {
		// The regression case: "redis" happens to also be a URL scheme
		// go-redis recognizes, so this must NOT go through ParseURL.
		opts, err := resolveOptions("redis:6379")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.Addr != "redis:6379" {
			t.Fatalf("Addr = %q, want %q", opts.Addr, "redis:6379")
		}
	})

	t.Run("bare localhost and port", func(t *testing.T) {
		opts, err := resolveOptions("localhost:6379")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.Addr != "localhost:6379" {
			t.Fatalf("Addr = %q, want %q", opts.Addr, "localhost:6379")
		}
	})

	t.Run("full redis URL with auth and db", func(t *testing.T) {
		opts, err := resolveOptions("redis://:secret@myhost:6380/2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.Addr != "myhost:6380" {
			t.Fatalf("Addr = %q, want %q", opts.Addr, "myhost:6380")
		}
		if opts.Password != "secret" {
			t.Fatalf("Password = %q, want %q", opts.Password, "secret")
		}
		if opts.DB != 2 {
			t.Fatalf("DB = %d, want 2", opts.DB)
		}
	})

	t.Run("malformed URL still errors", func(t *testing.T) {
		if _, err := resolveOptions("redis://%zz"); err == nil {
			t.Fatal("expected an error for a malformed URL, got nil")
		}
	})
}
