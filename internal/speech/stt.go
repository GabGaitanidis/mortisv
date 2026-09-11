package speech


type SttBridge interface {
	Listen() (string, error)
}
