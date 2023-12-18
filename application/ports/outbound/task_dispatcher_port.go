package outbound

type TaskDispatcher interface {
	Submit(task func()) error
}
