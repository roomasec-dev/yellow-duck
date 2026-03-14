package edr

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeDetailPayloadObject(t *testing.T) {
	payload, err := decodeDetailPayload(json.RawMessage(`{"k":"v","n":1}`), "incident view", "")
	if err != nil {
		t.Fatalf("decode object payload: %v", err)
	}
	if payload["k"] != "v" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestDecodeDetailPayloadJSONString(t *testing.T) {
	payload, err := decodeDetailPayload(json.RawMessage(`"{\"k\":\"v\"}"`), "incident view", "")
	if err != nil {
		t.Fatalf("decode string payload: %v", err)
	}
	if payload["k"] != "v" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestDecodeDetailPayloadPlainStringError(t *testing.T) {
	_, err := decodeDetailPayload(json.RawMessage(`"incident not found"`), "incident view", "ok")
	if err == nil {
		t.Fatal("expected error for plain string payload")
	}
	if !strings.Contains(err.Error(), "incident not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}
