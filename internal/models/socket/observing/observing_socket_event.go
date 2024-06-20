package observing

type ObservingSocketEvent struct {
	Event   string                 `json:"event"`
	Payload ObservingSocketPayload `json:"payload"`
}
