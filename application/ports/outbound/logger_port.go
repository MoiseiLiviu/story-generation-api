package outbound

type LoggerPort interface {
	Info(msg string)
	InfoWithFields(msg string, fields map[string]interface{})
	Error(err error, msg string)
	ErrorWithFields(err error, msg string, fields map[string]interface{})
	Debug(msg string)
	DebugWithFields(msg string, fields map[string]interface{})
	Warn(msg string)
	WarnWithFields(msg string, fields map[string]interface{})
}
