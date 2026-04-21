package crypto

import "testing"

func TestComputeAndValidateSignature(t *testing.T) {
	body := []byte(`{"chamado_id":"CH-1"}`)
	secret := "secret"
	signature := ComputeSignature(body, secret)

	if !ValidateSignature(body, signature, secret) {
		t.Fatalf("expected signature to validate")
	}
	if ValidateSignature(body, signature, "other-secret") {
		t.Fatalf("expected signature to fail with wrong secret")
	}
	if ValidateSignature(body, "deadbeef", secret) {
		t.Fatalf("expected signature without prefix to fail")
	}
}
