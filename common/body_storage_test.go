package common

import (
	"errors"
	"io"
	"testing"
)

func TestMemoryStorageNewReaderIsIndependent(t *testing.T) {
	storage := newMemoryStorage([]byte("payload"))
	t.Cleanup(func() { _ = storage.Close() })

	first, err := storage.NewReader()
	if err != nil {
		t.Fatalf("create first reader: %v", err)
	}
	defer first.Close()
	second, err := storage.NewReader()
	if err != nil {
		t.Fatalf("create second reader: %v", err)
	}
	defer second.Close()

	one := make([]byte, 3)
	if _, err := io.ReadFull(first, one); err != nil {
		t.Fatalf("read first reader prefix: %v", err)
	}
	if got := string(one); got != "pay" {
		t.Fatalf("first reader prefix = %q, want %q", got, "pay")
	}
	all, err := io.ReadAll(second)
	if err != nil {
		t.Fatalf("read second reader: %v", err)
	}
	if got := string(all); got != "payload" {
		t.Fatalf("second reader = %q, want full payload", got)
	}

	if err := storage.Close(); err != nil {
		t.Fatalf("close storage: %v", err)
	}
	if _, err := storage.NewReader(); !errors.Is(err, ErrStorageClosed) {
		t.Fatalf("NewReader after close error = %v, want ErrStorageClosed", err)
	}
}

func TestDiskStorageNewReaderIsIndependent(t *testing.T) {
	storage, err := newDiskStorage([]byte("disk-payload"), "")
	if err != nil {
		t.Fatalf("create disk storage: %v", err)
	}
	t.Cleanup(func() { _ = storage.Close() })

	first, err := storage.NewReader()
	if err != nil {
		t.Fatalf("create first reader: %v", err)
	}
	defer first.Close()
	second, err := storage.NewReader()
	if err != nil {
		t.Fatalf("create second reader: %v", err)
	}
	defer second.Close()

	one := make([]byte, 4)
	if _, err := io.ReadFull(first, one); err != nil {
		t.Fatalf("read first reader prefix: %v", err)
	}
	if got := string(one); got != "disk" {
		t.Fatalf("first reader prefix = %q, want %q", got, "disk")
	}
	all, err := io.ReadAll(second)
	if err != nil {
		t.Fatalf("read second reader: %v", err)
	}
	if got := string(all); got != "disk-payload" {
		t.Fatalf("second reader = %q, want full payload", got)
	}

	if err := storage.Close(); err != nil {
		t.Fatalf("close storage: %v", err)
	}
	if _, err := storage.NewReader(); !errors.Is(err, ErrStorageClosed) {
		t.Fatalf("NewReader after close error = %v, want ErrStorageClosed", err)
	}
}
