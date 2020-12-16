package impl

import (
	"context"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/remoting"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

func init() {
	remoting.SetProtocolHeaderHandler("dubbo3", NewTripleHeaderHandler)
}

type TripleHeader struct {
	method   string
	streamID uint32

	ContentType string

	ServiceVersion string
	ServiceGroup   string
	RPCID          string
	TracingID      string
	TracingRPCID   string
	TracingContext string
	ClusterInfo    string
}

func (t *TripleHeader) GetMethod() string {
	return t.method
}
func (t *TripleHeader) GetStreamID() uint32 {
	return t.streamID
}

func (t *TripleHeader) FieldToCtx() context.Context {
	ctx := context.WithValue(context.Background(), "tri-service-version", t.ServiceVersion)
	ctx = context.WithValue(ctx, "tri-service-group", t.ServiceGroup)
	ctx = context.WithValue(ctx, "tri-req-id", t.RPCID)
	ctx = context.WithValue(ctx, "tri-trace-traceid", t.TracingID)
	ctx = context.WithValue(ctx, "tri-trace-rpcid", t.TracingRPCID)
	ctx = context.WithValue(ctx, "tri-trace-proto-bin", t.TracingContext)
	ctx = context.WithValue(ctx, "tri-unit-info", t.ClusterInfo)
	return ctx
}

func NewTripleHeaderHandler() remoting.ProtocolHeaderHandler {
	return &TripleHeaderHandler{}
}

type TripleHeaderHandler struct {
}

// WriteHeaderField called when comsumer call server invoke,
// it parse field of url and ctx to HTTP2 Header field, developer must assure "tri-" prefix field be string
// if not, it will cause panic!
func (t TripleHeaderHandler) WriteHeaderField(url *common.URL, ctx context.Context) []hpack.HeaderField {
	headerFields := make([]hpack.HeaderField, 0, 2) // at least :status, content-type will be there if none else.
	headerFields = append(headerFields, hpack.HeaderField{Name: ":method", Value: "POST"})
	headerFields = append(headerFields, hpack.HeaderField{Name: ":scheme", Value: "http"})
	headerFields = append(headerFields, hpack.HeaderField{Name: ":path", Value: url.GetParam(":path", "")}) // added when invoke, parse grpc 'method' to :path
	headerFields = append(headerFields, hpack.HeaderField{Name: ":authority", Value: url.Location})
	headerFields = append(headerFields, hpack.HeaderField{Name: "content-type", Value: url.GetParam("content-type", "application/grpc")})
	headerFields = append(headerFields, hpack.HeaderField{Name: "user-agent", Value: "grpc-go/1.35.0-dev"})
	headerFields = append(headerFields, hpack.HeaderField{Name: "tri-service-version", Value: getCtxVaSave(ctx, "tri-service-version")})
	headerFields = append(headerFields, hpack.HeaderField{Name: "tri-service-group", Value: getCtxVaSave(ctx, "tri-service-group")})
	headerFields = append(headerFields, hpack.HeaderField{Name: "tri-req-id", Value: getCtxVaSave(ctx, "tri-req-id")})
	headerFields = append(headerFields, hpack.HeaderField{Name: "tri-trace-traceid", Value: getCtxVaSave(ctx, "tri-trace-traceid")})
	headerFields = append(headerFields, hpack.HeaderField{Name: "tri-trace-rpcid", Value: getCtxVaSave(ctx, "tri-trace-rpcid")})
	headerFields = append(headerFields, hpack.HeaderField{Name: "tri-trace-proto-bin", Value: getCtxVaSave(ctx, "tri-trace-proto-bin")})
	headerFields = append(headerFields, hpack.HeaderField{Name: "tri-unit-info", Value: getCtxVaSave(ctx, "tri-unit-info")})
	return headerFields
}

func getCtxVaSave(ctx context.Context, field string) string {
	val, ok := ctx.Value(field).(string)
	if ok {
		return val
	}
	return ""
}

func (t TripleHeaderHandler) ReadFromH2MetaHeader(frame *http2.MetaHeadersFrame) remoting.ProtocolHeader {
	tripleHeader := &TripleHeader{
		streamID: frame.StreamID,
	}
	for _, f := range frame.Fields {
		switch f.Name {
		case "tri-service-version":
			tripleHeader.ServiceVersion = f.Value
		case "tri-service-group":
			tripleHeader.ServiceGroup = f.Value
		case "tri-req-id":
			tripleHeader.RPCID = f.Value
		case "tri-trace-traceid":
			tripleHeader.TracingID = f.Value
		case "tri-trace-rpcid":
			tripleHeader.TracingRPCID = f.Value
		case "tri-trace-proto-bin":
			tripleHeader.TracingContext = f.Value
		case "tri-unit-info":
			tripleHeader.ClusterInfo = f.Value

		case "content-type":
			tripleHeader.ContentType = f.Value
			//contentSubtype, validContentType := grpcutil.ContentSubtype(f.Value)
			//if !validContentType {
			//	d.data.contentTypeErr = fmt.Sprintf("transport: received the unexpected content-type %q", f.Value)
			//	return
			//}
			//d.data.contentSubtype = contentSubtype
			//// TODO: do we want to propagate the whole content-type in the metadata,
			//// or come up with a way to just propagate the content-subtype if it was set?
			//// ie {"content-type": "application/grpc+proto"} or {"content-subtype": "proto"}
			//// in the metadata?
			//d.addMetadata(f.Name, f.Value)
			//d.data.isGRPC = true
		case "grpc-encoding":
			//d.data.encoding = f.Value
		case "grpc-status":
			//code, err := strconv.Atoi(f.Value)
			//if err != nil {
			//	d.data.grpcErr = status.Errorf(codes.Internal, "transport: malformed grpc-status: %v", err)
			//	return
			//}
			//d.data.rawStatusCode = &code
		case "grpc-message":
			//d.data.rawStatusMsg = decodeGrpcMessage(f.Value)
		case "grpc-status-details-bin":
			//v, err := decodeBinHeader(f.Value)
			//if err != nil {
			//	d.data.grpcErr = status.Errorf(codes.Internal, "transport: malformed grpc-status-details-bin: %v", err)
			//	return
			//}
			//s := &spb.Status{}
			//if err := proto.Unmarshal(v, s); err != nil {
			//	d.data.grpcErr = status.Errorf(codes.Internal, "transport: malformed grpc-status-details-bin: %v", err)
			//	return
			//}
			//d.data.statusGen = status.FromProto(s)
		case "grpc-timeout":
			//d.data.timeoutSet = true
			//var err error
			//if d.data.timeout, err = decodeTimeout(f.Value); err != nil {
			//	d.data.grpcErr = status.Errorf(codes.Internal, "transport: malformed time-out: %v", err)
			//}
		case ":path":
			tripleHeader.method = f.Value
		case ":status":
			//code, err := strconv.Atoi(f.Value)
			//if err != nil {
			//	d.data.httpErr = status.Errorf(codes.Internal, "transport: malformed http-status: %v", err)
			//	return
			//}
			//d.data.httpStatus = &code
		case "grpc-tags-bin":
			//v, err := decodeBinHeader(f.Value)
			//if err != nil {
			//	d.data.grpcErr = status.Errorf(codes.Internal, "transport: malformed grpc-tags-bin: %v", err)
			//	return
			//}
			//d.data.statsTags = v
			//d.addMetadata(f.Name, string(v))
		case "grpc-trace-bin":
			//v, err := decodeBinHeader(f.Value)
			//if err != nil {
			//	d.data.grpcErr = status.Errorf(codes.Internal, "transport: malformed grpc-trace-bin: %v", err)
			//	return
			//}
			//d.data.statsTrace = v
			//d.addMetadata(f.Name, string(v))
		default:
			//if isReservedHeader(f.Name) && !isWhitelistedHeader(f.Name) {
			//	break
			//}
			//v, err := decodeMetadataHeader(f.Name, f.Value)
			//if err != nil {
			//	if logger.V(logLevel) {
			//		logger.Errorf("Failed to decode metadata header (%q, %q): %v", f.Name, f.Value, err)
			//	}
			//	return
			//}
			//d.addMetadata(f.Name, v)
		}
	}
	return tripleHeader
}
