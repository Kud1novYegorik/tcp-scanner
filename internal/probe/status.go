package probe

type Status uint8

const (
	Unknown Status = iota
	Open
	Closed
	Timeout
	Unreachable
	Canceled
	Error
)

func (s Status) String() string {
	switch s {
	case Open:
		return "open"
	case Closed:
		return "closed"
	case Timeout:
		return "timeout"
	case Unreachable:
		return "unreachable"
	case Canceled:
		return "canceled"
	case Error:
		return "error"
	default:
		return "unknown"
	}
}
