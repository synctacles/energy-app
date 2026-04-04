package hasensor

import (
	"context"
	"log/slog"

	"github.com/synctacles/energy-app/internal/ha"
)

// RESTPublisher publishes sensor state via the HA Supervisor REST API.
// This always works when running as an HA addon with supervisor access.
type RESTPublisher struct {
	supervisor *ha.SupervisorClient
}

// NewRESTPublisher creates a REST-based sensor publisher.
func NewRESTPublisher(supervisor *ha.SupervisorClient) *RESTPublisher {
	return &RESTPublisher{supervisor: supervisor}
}

// UpdateSensor sets the state and attributes of a sensor entity via POST /core/api/states.
func (p *RESTPublisher) UpdateSensor(ctx context.Context, entityID, state string, attrs map[string]any) error {
	slog.Debug("publishing sensor", "entity", entityID, "state", state)
	return p.supervisor.PostState(ctx, entityID, state, attrs)
}
