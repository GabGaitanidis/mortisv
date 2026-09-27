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

func (c Command) GetStringSlice(key string) []string {
	if values, ok := c.Params[key].([]string); ok {
		return values
	}
	items, ok := c.Params[key].([]interface{})
	if !ok {
		return nil
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		if value, ok := item.(string); ok {
			values = append(values, value)
		}
	}
	return values
}
