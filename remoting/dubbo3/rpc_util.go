package dubbo3

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
)

// toRPCErr converts an error into an error from the status package.
func toRPCErr(err error) error {
	if err == nil || err == io.EOF {
		return err
	}
	if err == io.ErrUnexpectedEOF {
		return status.Error(codes.Internal, err.Error())
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	return status.Error(codes.Unknown, err.Error())
}
