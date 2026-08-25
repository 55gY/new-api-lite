package common

import (
	"io"
	"testing"
)

func TestNewOutboundJSONBodyProvidesReplayReaders(t *testing.T) {
	body, size, getBody, closer, err := NewOutboundJSONBody([]byte(`{"model":"test"}`))
	if err != nil {
		t.Fatalf("create outbound body: %v", err)
	}
	defer closer.Close()

	if want := int64(len(`{"model":"test"}`)); size != want {
		t.Fatalf("body size = %d, want %d", size, want)
	}
	primary, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read primary body: %v", err)
	}
	if got := string(primary); got != `{"model":"test"}` {
		t.Fatalf("primary body = %q", got)
	}

	for i := 0; i < 2; i++ {
		replay, err := getBody()
		if err != nil {
			t.Fatalf("create replay reader %d: %v", i, err)
		}
		payload, readErr := io.ReadAll(replay)
		closeErr := replay.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("read/close replay %d: read=%v close=%v", i, readErr, closeErr)
		}
		if got := string(payload); got != `{"model":"test"}` {
			t.Fatalf("replay body %d = %q", i, got)
		}
	}
}
