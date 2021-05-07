package commands

func (e Errors) ContainsErr(prop string, err error) bool {
	return e.ContainsStr(prop, err.Error())
}

func (e Errors) ContainsStr(prop string, err string) bool {
	messages, ok := e[prop]
	if !ok {
		return false
	}

	for _, message := range messages {
		if message.Error() == err {
			return true
		}
	}
	return false
}

func (e Errors) EmptyForProperty(prop string) bool {
	_, ok := e[prop]
	if !ok {
		return true
	}
	return false
}
