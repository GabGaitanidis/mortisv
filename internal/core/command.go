package core

// Command mirrors the Java class. Fields are exported (capitalized) so
// other packages and encoding/json can access them directly.
type Command struct {
	ActivityName string
	Module       string
	Action       string
	Params       map[string]interface{}
}

// Get returns a raw param value, same as Java's command.get(key).
func (c Command) Get(key string) interface{} {
	return c.Params[key]
}

// GetString is a convenience helper — Go has no automatic Object->String
// casting, so callers that expect a string param use this instead of a
// type assertion at every call site.
func (c Command) GetString(key string) string {
	v, ok := c.Params[key].(string)
	if !ok {
		return ""
	}
	return v
}
