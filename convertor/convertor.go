package convertor

import "github.com/mashenjun/mole/proto"

type IConvert interface {
	Convert() error
	GetSink() <-chan *proto.CSVMsg
}
