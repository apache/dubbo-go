# Serialization Extensions

Dubbo-go supports external serialization plugins registered by name through
`common/extension`. Applications select a plugin with
`client.WithSerialization` and `server.WithSerialization`. This guide covers
the public contracts and Triple integration.

Triple continues to manage HTTP headers, trailers, compression and framing.

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

An omitted provider receive list accepts the built-in Protobuf/JSON
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

Discovered Triple providers share their reference's codec binding.
Closing one connection does not close another connection's codec. Reconnection
uses the original local configuration snapshot. Triple services sharing a route
keep separate codec and receive-policy contexts for each group/version; requests
already executing retain their captured context when a service is replaced.

## Protocol Requirements

### Triple

Protobuf and ProtoJSON use message encoding. Hessian2, MsgPack and custom object
formats use wrappers. Wrapped-object streaming is unsupported and returns an
error; use message encoding for streaming services.

Primitive argument type names use the existing mappings. Invocation-declared
parameter types take precedence. A plugin accepting application structs without
declared types can implement `serialization.ArgumentTypeResolver` to supply
remote parameter class names without requiring Hessian POJO interfaces.

## Compatibility

The existing low-level Triple `Codec`, `WithCodec` and `WrapperCodec` APIs remain
available. Existing Protobuf, ProtoJSON, Hessian2 and MsgPack formats retain their
wire behavior. Registering a Go plugin does not establish cross-language
compatibility; peers must agree on the format and parameter type mapping.
