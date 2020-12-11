package impl

import (
	"github.com/apache/dubbo-go/remoting"
	"golang.org/x/net/http2"
)

func init() {
	remoting.SetProtocolHeaderHandler("dubbo3", NewTripleHeaderHandler)
}

type TripleHeaderHandler struct {
}

func (t TripleHeaderHandler) ReadFromH2MetaHeader(frame *http2.MetaHeadersFrame) remoting.ProtocolHeader {
	tripleHeader := &TripleHeader{
		streamID: frame.StreamID,
	}
	for _, f := range frame.Fields {
		switch f.Name {
		case "content-type":
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

type TripleHeader struct {
	method   string
	streamID uint32
}

func (t *TripleHeader) GetMethod() string {
	return t.method
}
func (t *TripleHeader) GetStreamID() uint32 {
	return t.streamID
}

func NewTripleHeaderHandler() remoting.ProtocolHeaderHandler {
	return &TripleHeaderHandler{}
}
