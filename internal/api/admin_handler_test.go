package api

import (
	"encoding/json"
	"strings"
	"testing"

	"monitor/internal/change"
	"monitor/internal/onboarding"
)

func TestAdminSensitiveKeyFieldsAreNeverSerialized(t *testing.T) {
	const ciphertext = "encrypted-key-material"

	submissionJSON, err := json.Marshal(onboarding.Submission{
		APIKeyEncrypted: ciphertext,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(submissionJSON), ciphertext) || strings.Contains(string(submissionJSON), "api_key_encrypted") {
		t.Fatalf("onboarding response leaked encrypted API Key: %s", submissionJSON)
	}

	changeJSON, err := json.Marshal(change.ChangeRequest{
		NewKeyEncrypted: ciphertext,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(changeJSON), ciphertext) || strings.Contains(string(changeJSON), "new_key_encrypted") {
		t.Fatalf("change response leaked encrypted API Key: %s", changeJSON)
	}
}
