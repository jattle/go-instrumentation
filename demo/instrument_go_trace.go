package demo

import (
	gonativectx "context"
	"encoding/json"
	"runtime/trace"
)

// ProcessFunc instrumentation function example using go trace
// usage: find . -name "*.go"|grep -vE "test|example"|xargs -I {} go-instrument-tool -source={} -replace -patches=xxx/demo/instrument_go_trace.go
func InstrumentGoTrace(spanName string, hasCtx bool, ctx gonativectx.Context, args ...interface{}) {
	fctx := gonativectx.TODO()
	var t *trace.Task
	if hasCtx {
		fctx = ctx
		ctx, t = trace.NewTask(fctx, spanName)
	} else {
		_, t = trace.NewTask(fctx, spanName)
	}
	logbin, _ := json.Marshal(args)
	trace.Logf(ctx, spanName, "function args: %s", string(logbin))
	defer t.End()
}
