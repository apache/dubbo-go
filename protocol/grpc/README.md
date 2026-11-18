# gRPC Serialization Extensions

The independent grpc-go protocol accepts serialization plugins through the
public `common/extension` registry. This protocol is separate from Triple's
gRPC-compatible wire format.

The public contracts and configuration snapshots are described in the
[serialization guide](../../serialization/README.md).

During `init()`, register a codec with `Protocol: "grpc"`, `Mode: serialization.MessageMode`, and
identical `Name` and `WireName` values. The codec implements
`serialization.Codec`; select its name through the existing serialization
configuration on both peers. The server dispatches incoming requests by their
content subtype while leaving standard Protobuf traffic available.

## Subtype Registration and Configuration

The independent gRPC protocol installs a stable grpc-go subtype adapter for a
`MessageMode` registration. `Name` and `WireName` must match. The existing
Protobuf and `jsonpb` JSON defaults remain available alongside custom subtypes.
A Triple wrapper registration does not register an independent gRPC subtype.

The subtype registry is process-wide. Reusing a subtype requires equivalent
read-only configuration; conflicting configuration or an existing external
registration returns an error. Factories receive no endpoint or side identity.
Only `Config` and namespaced `ConfigPrefix.*` parameters configure this binding;
use a separate subtype name for a different format or configuration.

Descriptors reserve stable subtype adapters during initialization, before any
RPCs start. Runtime configuration binds the codec instance to its reserved
adapter without changing grpc-go's global registry. A factory failure leaves
the adapter unconfigured and permits a later initialization retry. Incoming
requests for an unconfigured subtype return an error.

Starting a dubbo-go gRPC client or server ends descriptor registration for the
process. Register all custom gRPC descriptors during `init()`; later
registrations return an error without changing either registry. Plugin factories
and codec methods must be safe for concurrent use.
