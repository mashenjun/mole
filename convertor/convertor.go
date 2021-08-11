package convertor

type IConvert interface {
	// TODO: change the string to fs.File instead
	Convert() error
	GetSink() <-chan []string
}
