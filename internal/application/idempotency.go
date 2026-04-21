package application

import (
	"fmt"
	"time"

	"github.com/matheuss0xf/notifica-carioca/internal/domain"
)

func buildIdempotencyKey(event domain.WebhookEvent) string {
	return fmt.Sprintf("idemp:%s:%s:%s",
		event.ChamadoID, event.StatusNovo, event.Timestamp.Format(time.RFC3339Nano))
}
