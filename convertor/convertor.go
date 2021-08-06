package convertor

type IConvert interface {
	// TODO: change the string to fs.File instead
	Convert(string) error
	GetSink() <-chan []string
}
