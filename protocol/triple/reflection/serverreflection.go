/*
 *
 * Copyright 2016 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package reflection

import (
	"context"
	"io"
	"sort"
)

import (
	"google.golang.org/grpc/codes"

	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/internal"
	"dubbo.apache.org/dubbo-go/v3/internal/reflection"
	rpb "dubbo.apache.org/dubbo-go/v3/protocol/triple/reflection/triple_reflection"
	"dubbo.apache.org/dubbo-go/v3/server"
)

// ExtensionResolver is the interface used to query details about extensions.
// This interface is satisfied by protoregistry.GlobalTypes.
//
// # Experimental
//
// Notice: This type is EXPERIMENTAL and may be changed or removed in a
// later release.
type ExtensionResolver interface {
	protoregistry.ExtensionTypeResolver
	RangeExtensionsByMessage(message protoreflect.FullName, f func(protoreflect.ExtensionType) bool)
}

func NewServer() *ReflectionServer {
	return &ReflectionServer{
		descResolver: protoregistry.GlobalFiles,
		extResolver:  protoregistry.GlobalTypes,
	}
}

type ReflectionServer struct {
	s            reflection.ServiceInfoProvider
	descResolver protodesc.Resolver
	extResolver  ExtensionResolver
}

func (srv *ReflectionServer) Reference() string {
	return constant.ReflectionServiceTypeName
}

// fileDescWithDependencies returns a slice of serialized fileDescriptors in
// wire format ([]byte). The fileDescriptors will include fd and all the
// transitive dependencies of fd with names not in sentFileDescriptors.
func (s *ReflectionServer) fileDescWithDependencies(fd protoreflect.FileDescriptor, sentFileDescriptors map[string]bool) ([][]byte, error) {
	if fd.IsPlaceholder() {
		// If the given root file is a placeholder, treat it
		// as missing instead of serializing it.
		return nil, protoregistry.NotFound
	}
	var r [][]byte
	queue := []protoreflect.FileDescriptor{fd}
	for len(queue) > 0 {
		currentfd := queue[0]
		queue = queue[1:]
		if currentfd.IsPlaceholder() {
			// Skip any missing files in the dependency graph.
			continue
		}
		if sent := sentFileDescriptors[currentfd.Path()]; len(r) == 0 || !sent {
			sentFileDescriptors[currentfd.Path()] = true
			fdProto := protodesc.ToFileDescriptorProto(currentfd)
			currentfdEncoded, err := proto.Marshal(fdProto)
			if err != nil {
				return nil, err
			}
			r = append(r, currentfdEncoded)
		}
		for i := 0; i < currentfd.Imports().Len(); i++ {
			queue = append(queue, currentfd.Imports().Get(i))
		}
	}
	return r, nil
}

// fileDescEncodingContainingSymbol finds the file descriptor containing the
// given symbol, finds all of its previously unsent transitive dependencies,
// does marshaling on them, and returns the marshaled result. The given symbol
// can be a type, a service or a method.
func (s *ReflectionServer) fileDescEncodingContainingSymbol(name string, sentFileDescriptors map[string]bool) ([][]byte, error) {
	d, err := s.descResolver.FindDescriptorByName(protoreflect.FullName(name))
	if err != nil {
		return nil, err
	}
	return s.fileDescWithDependencies(d.ParentFile(), sentFileDescriptors)
}

// fileDescEncodingContainingExtension finds the file descriptor containing
// given extension, finds all of its previously unsent transitive dependencies,
// does marshaling on them, and returns the marshaled result.
func (s *ReflectionServer) fileDescEncodingContainingExtension(typeName string, extNum int32, sentFileDescriptors map[string]bool) ([][]byte, error) {
	xt, err := s.extResolver.FindExtensionByNumber(protoreflect.FullName(typeName), protoreflect.FieldNumber(extNum))
	if err != nil {
		return nil, err
	}
	return s.fileDescWithDependencies(xt.TypeDescriptor().ParentFile(), sentFileDescriptors)
}

// allExtensionNumbersForTypeName returns all extension numbers for the given type.
func (s *ReflectionServer) allExtensionNumbersForTypeName(name string) ([]int32, error) {
	var numbers []int32
	s.extResolver.RangeExtensionsByMessage(protoreflect.FullName(name), func(xt protoreflect.ExtensionType) bool {
		numbers = append(numbers, int32(xt.TypeDescriptor().Number()))
		return true
	})
	sort.Slice(numbers, func(i, j int) bool {
		return numbers[i] < numbers[j]
	})
	if len(numbers) == 0 {
		// maybe return an error if given type name is not known
		if _, err := s.descResolver.FindDescriptorByName(protoreflect.FullName(name)); err != nil {
			return nil, err
		}
	}
	return numbers, nil
}

// listServices returns the names of services this server exposes.
func (s *ReflectionServer) listServices() []*rpb.ServiceResponse {
	serviceInfo := s.s.GetServiceInfo()
	resp := make([]*rpb.ServiceResponse, 0, len(serviceInfo))
	for svc := range serviceInfo {
		resp = append(resp, &rpb.ServiceResponse{Name: svc})
	}
	sort.Slice(resp, func(i, j int) bool {
		return resp[i].Name < resp[j].Name
	})
	return resp
}

// ServerReflectionInfo is the reflection service handler.
func (s *ReflectionServer) ServerReflectionInfo(ctx context.Context, stream rpb.ServerReflection_ServerReflectionInfoServer) error {
	sentFileDescriptors := make(map[string]bool)
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		out := &rpb.ServerReflectionResponse{
			ValidHost:       in.Host,
			OriginalRequest: in,
		}
		switch req := in.MessageRequest.(type) {
		case *rpb.ServerReflectionRequest_FileByFilename:
			var b [][]byte
			fd, err := s.descResolver.FindFileByPath(req.FileByFilename)
			if err == nil {
				b, err = s.fileDescWithDependencies(fd, sentFileDescriptors)
			}
			if err != nil {
				out.MessageResponse = &rpb.ServerReflectionResponse_ErrorResponse{
					ErrorResponse: &rpb.ErrorResponse{
						ErrorCode:    int32(codes.NotFound),
						ErrorMessage: err.Error(),
					},
				}
			} else {
				out.MessageResponse = &rpb.ServerReflectionResponse_FileDescriptorResponse{
					FileDescriptorResponse: &rpb.FileDescriptorResponse{FileDescriptorProto: b},
				}
			}
		case *rpb.ServerReflectionRequest_FileContainingSymbol:
			b, err := s.fileDescEncodingContainingSymbol(req.FileContainingSymbol, sentFileDescriptors)
			if err != nil {
				out.MessageResponse = &rpb.ServerReflectionResponse_ErrorResponse{
					ErrorResponse: &rpb.ErrorResponse{
						ErrorCode:    int32(codes.NotFound),
						ErrorMessage: err.Error(),
					},
				}
			} else {
				out.MessageResponse = &rpb.ServerReflectionResponse_FileDescriptorResponse{
					FileDescriptorResponse: &rpb.FileDescriptorResponse{FileDescriptorProto: b},
				}
			}
		case *rpb.ServerReflectionRequest_FileContainingExtension:
			typeName := req.FileContainingExtension.ContainingType
			extNum := req.FileContainingExtension.ExtensionNumber
			b, err := s.fileDescEncodingContainingExtension(typeName, extNum, sentFileDescriptors)
			if err != nil {
				out.MessageResponse = &rpb.ServerReflectionResponse_ErrorResponse{
					ErrorResponse: &rpb.ErrorResponse{
						ErrorCode:    int32(codes.NotFound),
						ErrorMessage: err.Error(),
					},
				}
			} else {
				out.MessageResponse = &rpb.ServerReflectionResponse_FileDescriptorResponse{
					FileDescriptorResponse: &rpb.FileDescriptorResponse{FileDescriptorProto: b},
				}
			}
		case *rpb.ServerReflectionRequest_AllExtensionNumbersOfType:
			extNums, err := s.allExtensionNumbersForTypeName(req.AllExtensionNumbersOfType)
			if err != nil {
				out.MessageResponse = &rpb.ServerReflectionResponse_ErrorResponse{
					ErrorResponse: &rpb.ErrorResponse{
						ErrorCode:    int32(codes.NotFound),
						ErrorMessage: err.Error(),
					},
				}
			} else {
				out.MessageResponse = &rpb.ServerReflectionResponse_AllExtensionNumbersResponse{
					AllExtensionNumbersResponse: &rpb.ExtensionNumberResponse{
						BaseTypeName:    req.AllExtensionNumbersOfType,
						ExtensionNumber: extNums,
					},
				}
			}
		case *rpb.ServerReflectionRequest_ListServices:
			out.MessageResponse = &rpb.ServerReflectionResponse_ListServicesResponse{
				ListServicesResponse: &rpb.ListServiceResponse{
					Service: s.listServices(),
				},
			}
		default:
			return status.Errorf(codes.InvalidArgument, "invalid MessageRequest: %v", in.MessageRequest)
		}

		if err := stream.Send(out); err != nil {
			return err
		}
	}
}

var reflectionServer *ReflectionServer

func init() {
	reflectionServer = NewServer()
	internal.ReflectionRegister = Register
	server.SetProServices(&server.InternalService{
		Name: "reflection",
		Init: func(options *server.ServiceOptions) (*server.ServiceDefinition, bool) {
			return &server.ServiceDefinition{
				Handler: reflectionServer,
				Info:    &rpb.ServerReflection_ServiceInfo,
				Opts: []server.ServiceOption{server.WithNotRegister(),
					server.WithInterface(constant.ReflectionServiceInterface)},
			}, true
		},
		Priority: constant.DefaultPriority,
	})
	// In order to adapt config.Load
	// Plans for future removal
	config.SetProviderServiceWithInfo(reflectionServer, &rpb.ServerReflection_ServiceInfo)
}

func Register(s reflection.ServiceInfoProvider) {
	reflectionServer.s = s
}
