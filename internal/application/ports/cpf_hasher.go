package ports

// CPFHasher hashes citizen CPFs into deterministic opaque identifiers.
type CPFHasher interface {
	Hash(cpf string) string
}
