# Serialization and Remoting Codec Extensions

Dubbo-go supports serialization plugins registered by name through
`common/extension`. Applications can select a plugin with
`client.WithSerialization` and `server.WithSerialization`.

There are two separate extension points:

| Setting | Responsibility | Protocols |
| --- | --- | --- |
| `serialization` | Encode and decode messages or business objects | Triple, Dubbo, independent gRPC |
| `codec` | Encode and decode complete Dubbo request/response frames | Dubbo |

Triple continues to manage its HTTP headers, trailers, compression and framing.
Changing `serialization` does not replace those parts of the protocol.

## Getting Started

The following plugin encodes non-IDL Triple business objects as JSON with a
custom prefix:

```go
package taggedjson

import (
	"bytes"
	"encoding/json"
	"fmt"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/serialization"
)

type Codec struct{}

func (*Codec) Name() string { return "tagged-json" }

func (*Codec) Marshal(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return append([]byte("TAG:"), data...), nil
}

func (*Codec) Unmarshal(data []byte, target any) error {
	if !bytes.HasPrefix(data, []byte("TAG:")) {
		return fmt.Errorf("tagged-json: invalid prefix")
	}
	return json.Unmarshal(data[4:], target)
}

func init() {
	extension.MustRegisterSerialization(serialization.Registration{
		Protocol: "triple",
		Name:     "tagged-json",
		Mode:     serialization.WrappedObjectMode,
		WireName: "proto",
		New: func(serialization.Options) (serialization.Codec, error) {
			return &Codec{}, nil
		},
	})
}
```

Import the plugin package in both applications so that its registration runs.
For an initialized server, register the non-IDL service with the selected format:

```go
if err := srv.RegisterService(service,
	server.WithInterface("example.Echo"),
	server.WithSerialization("tagged-json"),
	server.WithAcceptedSerializations("tagged-json"),
); err != nil {
	return err
}
```

For an initialized client, use the same interface and format:

```go
conn, err := cli.Dial("example.Echo",
	client.WithIDL(constant.NONIDL),
	client.WithSerialization("tagged-json"),
)
if err != nil {
	return err
}
```

These application fragments use `client`, `server` and `common/constant` from
`dubbo.apache.org/dubbo-go/v3`. Configure discovery or a direct service URL as
usual, then use the returned connection for RPC calls.

The request encoding is `objects -> plugin -> TripleRequestWrapper -> protobuf`.
The wrapper carries the inner format name. The server checks its receive policy
before decoding and uses the same format for the response. The client checks the
response format before decoding it.

## Registration Modes

Registrations are identified by `(Protocol, Name)`:

| Protocol | Mode | Requirements |
| --- | --- | --- |
| `triple` | `MessageMode` | `WireName` equals `Name`; used for message encoding |
| `triple` | `WrappedObjectMode` | `WireName` is `proto`; used inside Triple wrappers |
| `dubbo` | `StreamMode` | Implements `StreamCodec` and supplies a legal `DubboID` |
| `grpc` | `MessageMode` | `WireName` equals `Name`; registered as a gRPC subtype |

Names must start with a lowercase letter or digit and contain only lowercase
letters, digits, dots, underscores or hyphens. Use canonical protocol names when
registering. URL protocol `tri` resolves to `triple` during lookup, and configured
serialization `protobuf` resolves to `proto`.

Duplicate registrations, nil factories, typed nil instances, mismatched instance
names and incompatible capabilities are rejected. `RegisterSerialization`
returns an error; `MustRegisterSerialization` panics on registration failure.

## Configuration

Reference/service settings override consumer/provider defaults. Select the
default format with `client.WithClientSerialization` or
`server.WithServerSerialization`, and the per-reference/service format with
`client.WithSerialization` or `server.WithSerialization`.

Use `WithClientAcceptedSerializations` and `WithServerAcceptedSerializations`
for defaults, or the corresponding `WithAcceptedSerializations` option on a
reference or service to set its receive policy.

The YAML settings belong to the corresponding consumer reference or provider
service. This configuration fragment shows the hierarchy for the Triple plugin:

```yaml
dubbo:
  protocols:
    triple:
      name: tri
      port: 20000
  provider:
    services:
      Echo:
        interface: example.Echo
        protocol-ids: [triple]
        serialization: tagged-json
        accepted-serializations: [tagged-json]
  consumer:
    references:
      Echo:
        interface: example.Echo
        protocol: tri
        serialization: tagged-json
        accepted-serializations: [tagged-json]
        protocol_config:
          triple:
            wire-protocol: grpc
```

Business service registration and plugin imports are still required. The
`wire-protocol` setting accepts `grpc` (default) or `triple`; both support the
wrapped-object plugin. To select Triple wire through the client API, pass
`client.WithClientProtocol(protocol.WithTriple(triple.WithWireProtocol("triple")))`
to `client.NewClient`. The `protocol` and `protocol/triple` packages provide
these protocol options.

For Triple, an omitted provider receive list accepts the built-in Protobuf/JSON
message formats, Hessian2 and the selected format. An explicit list is exact;
`accepted-serializations: []` allows no business formats. Health and reflection
handlers retain their separate Protobuf support. A client's wrapped response
default is its selected request format; an explicit list replaces that policy.
Unknown or disabled formats and malformed payloads return errors without trying
another business codec. Historical empty/`hessian4` wrapper aliases resolve to
`hessian2` and undergo the same receive-policy check.

### Plugin Configuration

Use the existing `extension.Config` and client/server `WithExtension` options.
Set `Registration.ConfigPrefix` to the configuration prefix and implement
`serialization.ConfigProvider` on the configuration. After `Init`, its
`CodecConfig()` method must return an independently owned, read-only snapshot.

The framework passes the snapshot as `serialization.Options.Config` and a
private copy of URL parameters as `Options.Parameters`. Configuration snapshots
stay local and are not published through service discovery. Without explicit
configuration, `Config` is nil; factories should provide defaults.

## Plugin Contracts and Lifecycle

Factories create instances during binding initialization, rather than per RPC.
Instances must support concurrent calls. `Marshal` and stream `Finish` return
independently owned bytes; `Unmarshal` must not retain its input after returning.
The optional `MarshalAppender` must preserve the destination prefix. Invalid
input should return an error.

Codecs may implement `io.Closer`. The framework releases created instances on
initialization failure and closes shared resources after their last user releases
them. Triple tracks active requests and streams before releasing resources;
applications must close streaming responses. A wrapping plugin that owns its
delegate must forward `Close` to it.

Discovered Triple and Dubbo connections share their reference's codec binding.
Closing one connection does not close another connection's codec. Reconnection
uses the original local configuration snapshot. Triple services sharing a route
keep separate codec and receive-policy contexts for each group/version; requests
already executing retain their captured context when a service is replaced.
Independent gRPC subtypes have process lifetime.

## Protocol Requirements

### Triple

Protobuf and ProtoJSON use message encoding. Hessian2, MsgPack and custom object
formats use wrappers. Wrapped-object streaming is unsupported and returns an
error; use message encoding for streaming services.

Primitive argument type names use the existing mappings. Invocation-declared
parameter types take precedence. A plugin accepting application structs without
declared types can implement `serialization.ArgumentTypeResolver` to supply
remote parameter class names without requiring Hessian POJO interfaces.

### Classic Dubbo

A stream registration must implement `serialization.StreamCodec` and provide a
nonzero 5-bit `DubboID`. IDs 2 and 21 are reserved for Hessian2 and Protobuf.
ID 21 has no built-in body implementation; selecting it requires a plugin.
Agree on custom IDs and wire formats with the peer.

Each encoder or decoder owns one complete message body, including protocol
fields, arguments, reference tables and attachments. Do not concatenate
independent `Marshal` results to emulate a stream. Stream `Close` must release
its resources, including on error paths.

`Decoder.Decode` must fill the supplied pointer using its concrete target type.
Metadata uses `*string`, response flags use `*int32`, and attachments use
`*map[string]any`. Arguments initially use `*any`; once the method is known,
arguments with mismatched dynamic types are decoded again from the original body
into concrete method argument pointers. Support both dynamic and concrete
targets, including numeric range validation. Response values use the caller's
concrete reply pointer. Plugins do not need to reproduce Hessian's dynamic map
or number types. Custom-format business exceptions use a string value.

### Dubbo Framing

Register a complete framing implementation separately with
`extension.RegisterCodec` or `extension.MustRegisterCodec`:

```go
func registerFraming(factory remoting.CodecFactory) error {
	return extension.RegisterCodec(remoting.CodecRegistration{
		Protocol: "dubbo",
		Name:     "my-framing",
		New:      factory,
	})
}
```

The factory receives a cloned `*common.URL` and returns a `remoting.Codec` or an
initialization error. Select the registered name on both endpoints:

```go
srv, err := server.NewServer(
	server.WithServerProtocol(
		protocol.WithDubbo(),
		protocol.WithID("dubbo"),
		protocol.WithCodec("my-framing"),
	),
	server.WithServerProtocolIDs([]string{"dubbo"}),
)
if err != nil {
	return err
}

cli, err := client.NewClient(
	client.WithClientProtocolDubbo(),
	client.WithClientProtocol(
		protocol.WithDubbo(),
		protocol.WithCodec("my-framing"),
	),
)
if err != nil {
	return err
}
```

In YAML, set `codec` on the provider's named protocol and `protocol_config.codec`
on the consumer reference. Without a custom name, the built-in Dubbo framing is
used. Instances are shared across sessions and must support concurrent calls;
partial-frame state belongs in the caller's input buffer.

Services sharing one Dubbo listener must use the same receive set, framing
selection and configuration scope. Incompatible settings fail during export;
use separate endpoints for them.

### Independent gRPC

The independent gRPC protocol installs a stable grpc-go subtype adapter for a
`MessageMode` registration. `Name` and `WireName` must match. The existing
Protobuf and `jsonpb` JSON defaults remain available alongside custom subtypes.
A Triple wrapper registration does not register an independent gRPC subtype.

The subtype registry is process-wide. Reusing a subtype requires equivalent
read-only configuration; conflicting configuration or an existing external
registration returns an error. Factories receive no endpoint or side identity.
Only `Config` and namespaced `ConfigPrefix.*` parameters configure this binding;
use a separate subtype name for a different format or configuration.

## Compatibility

The existing low-level Triple `Codec`, `WithCodec` and `WrapperCodec` APIs remain
available. Dubbo's legacy `impl.Serializer`, `SetSerializer`, `GetSerializerById`
and `LoadSerializer` APIs retain their compatibility behavior, as do remoting's
`RegistryCodec` and `GetCodec` singleton APIs.

Registering a Go plugin does not establish cross-language compatibility. Both
peers must agree on the format, parameter type mapping and, for Dubbo streams,
serialization ID.
