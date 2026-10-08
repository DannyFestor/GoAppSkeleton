package httpx

// StatusError is a handler error that the client caused, so its status and message are safe to show.
type StatusError struct {
	Status  int
	Message string
}

func (e StatusError) Error() string {
	return e.Message
}
