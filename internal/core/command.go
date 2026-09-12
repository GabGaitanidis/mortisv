package core

type Command struct {
	ActivityName string
	Module       string
	Action       string
	Params       map[string]interface{}
}

func (c Command) Get(key string) interface{} {
	return c.Params[key]
}

func (c Command) GetString(key string) string {
	v, ok := c.Params[key].(string)
	if !ok {
		return ""
	}
	return v
}
