# Dubbo Serialization and Remoting Codec Extensions

Dubbo supports two independent extension points: `serialization` selects a
message-body serializer, while `codec` selects complete request/response framing.
Register plugins through `common/extension`, then select their names on both
endpoints using the client/server configuration.

The public serialization contracts and configuration snapshot lifecycle are
described in the [serialization guide](../../serialization/README.md).

For a body plugin, register `Protocol: "dubbo"`, `Mode: serialization.StreamMode`
and the agreed `DubboID`, and return a `serialization.StreamCodec` from its factory.
Select it with `client.WithSerialization` and `server.WithSerialization`.
An explicit `accepted-serializations` list restricts the IDs accepted by the
binding. Omitting it preserves the default receive behavior.

## Body Serialization

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

## Remoting Codec

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

## Resource Ownership and Compatibility

Connections share codec bindings through independent leases. Closing or failing
one connection does not release a sibling's active codec. Final release closes
the shared instance; reconnection constructs a fresh generation from the
original local configuration. Services sharing a compatible listener reuse its
binding, which is released on listener shutdown.

The legacy `impl.Serializer`, `SetSerializer`, `GetSerializerById`, `LoadSerializer`,
`RegistryCodec` and `GetCodec` APIs retain their compatibility behavior.
Custom Go serializers do not establish Java interoperability: peers must agree
on the serialization ID, body format and parameter type mappings.
