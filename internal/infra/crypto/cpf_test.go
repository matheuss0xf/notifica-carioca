package crypto

import "testing"

func TestCPFHasherUsesKeyedDeterministicHash(t *testing.T) {
	cpf := "52998224725"
	first := NewCPFHasher("key-a").Hash(cpf)
	second := NewCPFHasher("key-a").Hash(cpf)
	third := NewCPFHasher("key-b").Hash(cpf)

	if first != second {
		t.Fatalf("expected deterministic hash for same key")
	}
	if first == third {
		t.Fatalf("expected different hash for different key")
	}
	if first != HashCPF(cpf, "key-a") {
		t.Fatalf("expected hasher to delegate to HashCPF")
	}
}
