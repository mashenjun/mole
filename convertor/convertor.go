package convertor

import (
	"context"
	"github.com/mashenjun/mole/proto"
)

type IConvert interface {
	Convert(context.Context) error
	GetSink() <-chan *proto.CSVMsg
}
