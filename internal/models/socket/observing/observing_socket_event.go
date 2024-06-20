package observing

type ObservingSocketEvent struct {
	Event          string          `json:"event"`
	Payload        json.RawMessage `json:"payload"`
}