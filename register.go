package kafka

import (
	"github.com/mstoykov/xk6-segment/pkg/segment"
	"go.k6.io/k6/js/modules"
)

func init() {
	modules.Register("k6/x/segment", segment.New())
}
