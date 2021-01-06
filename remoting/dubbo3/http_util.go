package dubbo3

import (
	"bytes"
	status2 "github.com/apache/dubbo-go/remoting/dubbo3/status"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"strconv"
	"strings"
)


import (
	"fmt"
	"unicode/utf8"
)



var (
	clientPreface   = []byte(http2.ClientPreface)
	http2ErrConvTab = map[http2.ErrCode]codes.Code{
		http2.ErrCodeNo:                 codes.Internal,
		http2.ErrCodeProtocol:           codes.Internal,
		http2.ErrCodeInternal:           codes.Internal,
		http2.ErrCodeFlowControl:        codes.ResourceExhausted,
		http2.ErrCodeSettingsTimeout:    codes.Internal,
		http2.ErrCodeStreamClosed:       codes.Internal,
		http2.ErrCodeFrameSize:          codes.Internal,
		http2.ErrCodeRefusedStream:      codes.Unavailable,
		http2.ErrCodeCancel:             codes.Canceled,
		http2.ErrCodeCompression:        codes.Internal,
		http2.ErrCodeConnect:            codes.Internal,
		http2.ErrCodeEnhanceYourCalm:    codes.ResourceExhausted,
		http2.ErrCodeInadequateSecurity: codes.PermissionDenied,
		http2.ErrCodeHTTP11Required:     codes.Internal,
	}
	// HTTPStatusConvTab is the HTTP status code to gRPC error code conversion table.
	HTTPStatusConvTab = map[int]codes.Code{
		// 400 Bad Request - INTERNAL.
		http.StatusBadRequest: codes.Internal,
		// 401 Unauthorized  - UNAUTHENTICATED.
		http.StatusUnauthorized: codes.Unauthenticated,
		// 403 Forbidden - PERMISSION_DENIED.
		http.StatusForbidden: codes.PermissionDenied,
		// 404 Not Found - UNIMPLEMENTED.
		http.StatusNotFound: codes.Unimplemented,
		// 429 Too Many Requests - UNAVAILABLE.
		http.StatusTooManyRequests: codes.Unavailable,
		// 502 Bad Gateway - UNAVAILABLE.
		http.StatusBadGateway: codes.Unavailable,
		// 503 Service Unavailable - UNAVAILABLE.
		http.StatusServiceUnavailable: codes.Unavailable,
		// 504 Gateway timeout - UNAVAILABLE.
		http.StatusGatewayTimeout: codes.Unavailable,
	}
)



const (
	spaceByte   = ' '
	tildeByte   = '~'
	percentByte = '%'
)




// decodeState configures decoding criteria and records the decoded data.
type decodeState struct {
	// whether decoding on server side or not
	serverSide bool

	// Records the states during HPACK decoding. It will be filled with info parsed from HTTP HEADERS
	// frame once decodeHeader function has been invoked and returned.
	data parsedHeaderData
}


func (d *decodeState) status() *status2.Status {
	if d.data.statusGen == nil {
		// No status-details were provided; generate status using code/msg.
		d.data.statusGen = status2.New(codes.Code(int32(*(d.data.rawStatusCode))), d.data.rawStatusMsg)
	}
	return d.data.statusGen
}


func (d *decodeState) processHeaderField(f hpack.HeaderField) {
	switch f.Name {
	case "grpc-status":
		code, err := strconv.Atoi(f.Value)
		if err != nil {
			d.data.grpcErr = status.Errorf(codes.Internal, "transport: malformed grpc-status: %v", err)
			return
		}
		d.data.rawStatusCode = &code
	case "grpc-message":
		d.data.rawStatusMsg = decodeGrpcMessage(f.Value)

	default:

	}
}




// constructErrMsg constructs error message to be returned in HTTP fallback mode.
// Format: HTTP status code and its corresponding message + content-type error message.
func (d *decodeState) constructHTTPErrMsg() string {
	var errMsgs []string

	if d.data.httpStatus == nil {
		errMsgs = append(errMsgs, "malformed header: missing HTTP status")
	} else {
		errMsgs = append(errMsgs, fmt.Sprintf("%s: HTTP status code %d", http.StatusText(*(d.data.httpStatus)), *d.data.httpStatus))
	}

	if d.data.contentTypeErr == "" {
		errMsgs = append(errMsgs, "transport: missing content-type field")
	} else {
		errMsgs = append(errMsgs, d.data.contentTypeErr)
	}

	return strings.Join(errMsgs, "; ")
}



// decodeGrpcMessage decodes the msg encoded by encodeGrpcMessage.
func decodeGrpcMessage(msg string) string {
	if msg == "" {
		return ""
	}
	lenMsg := len(msg)
	for i := 0; i < lenMsg; i++ {
		if msg[i] == percentByte && i+2 < lenMsg {
			return decodeGrpcMessageUnchecked(msg)
		}
	}
	return msg
}

func decodeGrpcMessageUnchecked(msg string) string {
	var buf bytes.Buffer
	lenMsg := len(msg)
	for i := 0; i < lenMsg; i++ {
		c := msg[i]
		if c == percentByte && i+2 < lenMsg {
			parsed, err := strconv.ParseUint(msg[i+1:i+3], 16, 8)
			if err != nil {
				buf.WriteByte(c)
			} else {
				buf.WriteByte(byte(parsed))
				i += 2
			}
		} else {
			buf.WriteByte(c)
		}
	}
	return buf.String()
}



type parsedHeaderData struct {
	encoding string
	// statusGen caches the stream status received from the trailer the server
	// sent.  Client side only.  Do not access directly.  After all trailers are
	// parsed, use the status method to retrieve the status.
	statusGen *status2.Status
	rawStatusCode *int
	rawStatusMsg  string
	httpStatus    *int
	isGRPC         bool
	grpcErr        error
	httpErr        error
	contentTypeErr string
}



func encodeGrpcMessage(msg string) string {
	if msg == "" {
		return ""
	}
	lenMsg := len(msg)
	for i := 0; i < lenMsg; i++ {
		c := msg[i]
		if !(c >= spaceByte && c <= tildeByte && c != percentByte) {
			return encodeGrpcMessageUnchecked(msg)
		}
	}
	return msg
}

func encodeGrpcMessageUnchecked(msg string) string {
	var buf bytes.Buffer
	for len(msg) > 0 {
		r, size := utf8.DecodeRuneInString(msg)
		for _, b := range []byte(string(r)) {
			if size > 1 {
				// If size > 1, r is not ascii. Always do percent encoding.
				buf.WriteString(fmt.Sprintf("%%%02X", b))
				continue
			}

			// The for loop is necessary even if size == 1. r could be
			// utf8.RuneError.
			//
			// fmt.Sprintf("%%%02X", utf8.RuneError) gives "%FFFD".
			if b >= spaceByte && b <= tildeByte && b != percentByte {
				buf.WriteByte(b)
			} else {
				buf.WriteString(fmt.Sprintf("%%%02X", b))
			}
		}
		msg = msg[size:]
	}
	return buf.String()
}




