package dubbo

import (
	invocation_impl "github.com/apache/dubbo-go/protocol/invocation"
	"github.com/opentracing/opentracing-go"
)

func injectTraceCtx(currentSpan opentracing.Span, inv *invocation_impl.RPCInvocation) error {
	// inject opentracing ctx
	traceAttachments := filterContext(inv.Attachments())
	carrier := opentracing.TextMapCarrier(traceAttachments)
	err := opentracing.GlobalTracer().Inject(currentSpan.Context(), opentracing.TextMap, carrier)
	if err == nil {
		fillTraceAttachments(inv.Attachments(), traceAttachments)
	}
	return err
}

func extractTraceCtx(inv *invocation_impl.RPCInvocation) (opentracing.SpanContext, error) {
	traceAttachments := filterContext(inv.Attachments())
	// actually, if user do not use any opentracing framework, the err will not be nil.
	spanCtx, err := opentracing.GlobalTracer().Extract(opentracing.TextMap,
		opentracing.TextMapCarrier(traceAttachments))
	return spanCtx, err
}

func filterContext(attachments map[string]interface{}) map[string]string {
	var traceAttchment = make(map[string]string)
	for k, v := range attachments {
		if r, ok := v.(string); ok {
			traceAttchment[k] = r
		}
	}
	return traceAttchment
}

func fillTraceAttachments(attachments map[string]interface{}, traceAttachment map[string]string) {
	for k, v := range traceAttachment {
		attachments[k] = v
	}
}
