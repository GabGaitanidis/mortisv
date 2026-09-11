package speech

// TtsBridge mirrors the Java interface of the same name. The concrete
// implementation will write to the Python TTS process's stdin over the
// persistent IPC pipe, same as today.
type TtsBridge interface {
	Speak(text string) error
}
