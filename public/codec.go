package public

//////////////////////////////////////////
// codec type
//////////////////////////////////////////

type CodecType int

const (
	CODECTYPE_UNKNOWN CodecType = iota
	CODECTYPE_JSONRPC
)

var codecTypeStrings = [...]string{
	"unknown",
	"jsonrpc",
}

func (c CodecType) String() string {
	typ := CODECTYPE_UNKNOWN
	switch c {
	case CODECTYPE_JSONRPC:
		typ = c
	}

	return codecTypeStrings[typ]
}

func GetCodecType(t string) CodecType {
	var typ = CODECTYPE_UNKNOWN

	switch t {
	case codecTypeStrings[CODECTYPE_JSONRPC]:
		typ = CODECTYPE_JSONRPC
	}

	return typ
}
