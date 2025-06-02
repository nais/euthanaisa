package resource

type Resource interface {
	GetGroup() string
	GetVersion() string
	GetName() string
}
