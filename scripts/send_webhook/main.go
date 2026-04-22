package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	cryptoutil "github.com/matheuss0xf/notifica-carioca/internal/infra/crypto"
)

type webhookPayload struct {
	ChamadoID      string    `json:"chamado_id"`
	Tipo           string    `json:"tipo"`
	CPF            string    `json:"cpf"`
	StatusAnterior string    `json:"status_anterior,omitempty"`
	StatusNovo     string    `json:"status_novo"`
	Titulo         string    `json:"titulo"`
	Descricao      string    `json:"descricao,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	baseURL := os.Getenv("APP_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	secret := os.Getenv("WEBHOOK_SECRET")
	if secret == "" {
		secret = "dev-webhook-secret"
	}

	payload := webhookPayload{
		ChamadoID:      "CH-2024-001234",
		Tipo:           "status_change",
		CPF:            "529.982.247-25",
		StatusAnterior: "em_analise",
		StatusNovo:     "em_execucao",
		Titulo:         "Buraco na Rua - Atualizacao",
		Descricao:      "Equipe designada para reparo na Rua das Laranjeiras, 100",
		Timestamp:      time.Now().UTC().Truncate(time.Second),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		logger.Error("marshal payload", "error", err)
		os.Exit(1)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/webhooks/status-change", bytes.NewReader(body))
	if err != nil {
		logger.Error("build request", "error", err)
		os.Exit(1)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-256", cryptoutil.ComputeSignature(body, secret))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error("send request", "error", err)
		os.Exit(1)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.Warn("close response body", "error", closeErr)
		}
	}()

	fmt.Printf("status=%s\n", resp.Status)
}
