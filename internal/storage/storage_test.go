package storage_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"transport-app/internal/storage"
)

type cfg struct {
	driver string
	dir    string
}

func (c *cfg) GetDriver() string   { return c.driver }
func (c *cfg) GetLocalDir() string { return c.dir }

func TestLocal_SaveOpenDelete(t *testing.T) {
	s, err := storage.New(&cfg{driver: "local", dir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()

	key, err := s.Save(ctx, "logos/acme/logo.png", strings.NewReader("imgbytes"), "image/png")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if key != "logos/acme/logo.png" {
		t.Errorf("returned key = %q", key)
	}

	rc, err := s.Open(ctx, "logos/acme/logo.png")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got, _ := io.ReadAll(rc)
	_ = rc.Close()
	if string(got) != "imgbytes" {
		t.Errorf("roundtrip = %q, want imgbytes", got)
	}

	if err := s.Delete(ctx, "logos/acme/logo.png"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	// Deleting a missing key is not an error.
	if err := s.Delete(ctx, "logos/acme/logo.png"); err != nil {
		t.Errorf("Delete(missing) = %v, want nil", err)
	}
}

func TestLocal_PathTraversalRejected(t *testing.T) {
	s, err := storage.New(&cfg{driver: "local", dir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for _, key := range []string{"../../etc/passwd", "..\\..\\win", "/../escape", "."} {
		if _, err := s.Save(context.Background(), key, strings.NewReader("x"), "text/plain"); err == nil {
			t.Errorf("Save(%q) = nil error, want traversal rejection", key)
		}
		if _, err := s.Open(context.Background(), key); err == nil {
			t.Errorf("Open(%q) = nil error, want traversal rejection", key)
		}
	}
}

func TestNew_UnsupportedAndS3NotWired(t *testing.T) {
	if _, err := storage.New(&cfg{driver: "gcs"}); err == nil {
		t.Error("unsupported driver accepted")
	}
	// s3 must fail loudly until a client is wired — never silently no-op.
	if _, err := storage.New(&cfg{driver: "s3"}); err == nil {
		t.Error("s3 driver returned nil error; expected explicit not-wired error")
	}
}
