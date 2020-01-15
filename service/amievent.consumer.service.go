package service

type AmiEventConsumer interface {
	Initialize() error
	Destroy()
	Consume(event map[string]string)
}
