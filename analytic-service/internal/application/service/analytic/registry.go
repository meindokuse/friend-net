package analytic

import (
	"github.com/meindokuse/cloud-drive/analytic-service/internal/application/service/analytic/delete_event"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/application/service/analytic/get_stats"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/application/service/analytic/ingest_event"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/application/service/analytic/list_events"
)

type Registry struct {
	IngestEvent *ingest_event.Service
	GetStats    *get_stats.Service
	ListEvents  *list_events.Service
	DeleteEvent *delete_event.Service
}
