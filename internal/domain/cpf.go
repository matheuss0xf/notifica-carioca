package domain

import "strings"

// NormalizeCPF removes any non-digit characters from a CPF string.
func NormalizeCPF(raw string) string {
	var b strings.Builder
	b.Grow(len(raw))

	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}

	return b.String()
}

// ValidateCPF normalizes and validates a CPF, returning only its digits.
func ValidateCPF(raw string) (string, error) {
	cpf := NormalizeCPF(raw)
	if len(cpf) != 11 {
		return "", ErrInvalidCPF
	}

	allSame := true
	for i := 1; i < len(cpf); i++ {
		if cpf[i] != cpf[0] {
			allSame = false
			break
		}
	}
	if allSame {
		return "", ErrInvalidCPF
	}

	if cpf[9] != calculateCPFCheckDigit(cpf[:9], 10) {
		return "", ErrInvalidCPF
	}
	if cpf[10] != calculateCPFCheckDigit(cpf[:10], 11) {
		return "", ErrInvalidCPF
	}

	return cpf, nil
}

func calculateCPFCheckDigit(base string, weight int) byte {
	sum := 0
	for _, r := range base {
		sum += int(r-'0') * weight
		weight--
	}

	remainder := sum % 11
	if remainder < 2 {
		return '0'
	}

	return byte('0' + (11 - remainder))
}
