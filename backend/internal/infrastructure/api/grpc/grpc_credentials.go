package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/smilemakc/mbflow/api/proto/serviceapipb"
	"github.com/smilemakc/mbflow/internal/application/serviceapi"
)

func (s *ServiceAPIServer) ListCredentials(ctx context.Context, req *pb.ListCredentialsRequest) (*pb.ListCredentialsResponse, error) {
	result, err := s.ops.ListCredentials(ctx, serviceapi.ListCredentialsParams{
		UserID:   req.UserId,
		Provider: req.Provider,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.ListCredentialsResponse{
		Credentials: toProtoCredentials(result.Credentials),
	}, nil
}

func (s *ServiceAPIServer) CreateCredential(ctx context.Context, req *pb.CreateCredentialRequest) (*pb.CredentialResponse, error) {
	userID, ok := UserIDFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	cred, err := s.ops.CreateCredential(ctx, serviceapi.CreateCredentialParams{
		UserID:         userID,
		Name:           req.Name,
		Description:    req.Description,
		CredentialType: req.CredentialType,
		Provider:       req.Provider,
		Data:           req.Data,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.CredentialResponse{
		Credential: toProtoCredential(cred),
	}, nil
}

func (s *ServiceAPIServer) UpdateCredential(ctx context.Context, req *pb.UpdateCredentialRequest) (*pb.CredentialResponse, error) {
	cred, err := s.ops.UpdateCredential(ctx, serviceapi.UpdateCredentialParams{
		CredentialID: req.Id,
		Name:         req.Name,
		Description:  req.Description,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.CredentialResponse{
		Credential: toProtoCredential(cred),
	}, nil
}

func (s *ServiceAPIServer) DeleteCredential(ctx context.Context, req *pb.DeleteCredentialRequest) (*pb.DeleteResponse, error) {
	if err := s.ops.DeleteCredential(ctx, serviceapi.DeleteCredentialParams{
		CredentialID: req.Id,
	}); err != nil {
		return nil, mapError(err)
	}

	return &pb.DeleteResponse{Message: "credential deleted successfully"}, nil
}
