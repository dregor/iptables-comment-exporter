package collect

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fakeBin(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-tables-save")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunOK(t *testing.T) {
	bin := fakeBin(t, `echo "hello"`)
	data, err := Run(context.Background(), bin, time.Second)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if string(data) != "hello\n" {
		t.Errorf("неверный вывод: %q", data)
	}
}

func TestRunTimeout(t *testing.T) {
	bin := fakeBin(t, `sleep 5`)
	if _, err := Run(context.Background(), bin, 100*time.Millisecond); err == nil {
		t.Fatal("ожидалась ошибка таймаута")
	}
}

func TestRunOversized(t *testing.T) {
	bin := fakeBin(t, `head -c $((33*1024*1024)) /dev/zero`)
	if _, err := Run(context.Background(), bin, 5*time.Second); err == nil {
		t.Fatal("ожидалась ошибка превышения лимита вывода")
	}
}
