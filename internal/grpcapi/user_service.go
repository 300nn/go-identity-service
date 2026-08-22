package grpcapi

import (
	"context"
	"errors"

	"CrudTutorialProject/internal/apperror"
	userapiv1 "CrudTutorialProject/internal/gen/api/user/v1"
	"CrudTutorialProject/internal/user"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserReader interface {
	GetUser(ctx context.Context, id int64) (user.User, error)
	ListUsers(ctx context.Context, input user.ListUsersInput) (user.ListUsersOutput, error)
}

type UserService struct {
	userapiv1.UnimplementedUserServiceServer

	users UserReader
}

func NewUserService(users UserReader) *UserService {
	return &UserService{
		users: users,
	}
}

func (s *UserService) GetUser(ctx context.Context, req *userapiv1.GetUserRequest) (*userapiv1.GetUserResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if err := requireSelfOrRole(ctx, req.GetId(), user.RoleAdmin); err != nil {
		return nil, err
	}

	usr, err := s.users.GetUser(ctx, req.GetId())

	if err != nil {
		return nil, toGRPCError(err)
	}

	return &userapiv1.GetUserResponse{
		User: userToProto(usr),
	}, nil
}

func (s *UserService) ListUsers(ctx context.Context, req *userapiv1.ListUsersRequest) (*userapiv1.ListUsersResponse, error) {
	if err := requireRole(ctx, user.RoleAdmin); err != nil {
		return nil, err
	}

	limit := req.GetLimit()

	if limit == 0 {
		limit = 20
	}

	if limit < 0 || limit > 100 {
		return nil, status.Error(codes.InvalidArgument, "limit must be between 1 and 100")
	}

	if req.GetOffset() < 0 {
		return nil, status.Error(codes.InvalidArgument, "offset must be greater than or equal to 0")
	}

	list, err := s.users.ListUsers(ctx, user.ListUsersInput{
		Limit:  int(limit),
		Offset: int(req.GetOffset()),
		Email:  req.GetEmail(),
		Sort:   req.GetSort(),
	})

	if err != nil {
		return nil, toGRPCError(err)
	}

	users := make([]*userapiv1.User, 0, len(list.Users))

	for _, usr := range list.Users {
		users = append(users, userToProto(usr))
	}

	return &userapiv1.ListUsersResponse{
		Users: users,
		Total: list.Total,
	}, nil
}

func userToProto(usr user.User) *userapiv1.User {
	return &userapiv1.User{
		Id:        usr.ID,
		Name:      usr.Name,
		Email:     usr.Email,
		Age:       int32(usr.Age),
		Role:      string(usr.Role),
		CreatedAt: timestamppb.New(usr.CreatedAt),
		UpdatedAt: timestamppb.New(usr.UpdatedAt),
	}
}

func toGRPCError(err error) error {
	var publicErr *apperror.PublicError
	if errors.As(err, &publicErr) {
		switch publicErr.Status {
		case 400:
			return status.Error(codes.InvalidArgument, publicErr.Message)
		case 401:
			return status.Error(codes.Unauthenticated, publicErr.Message)
		case 403:
			return status.Error(codes.PermissionDenied, publicErr.Message)
		case 404:
			return status.Error(codes.NotFound, publicErr.Message)
		case 409:
			return status.Error(codes.AlreadyExists, publicErr.Message)
		default:
			return status.Error(codes.Internal, publicErr.Message)
		}
	}

	return status.Error(codes.Internal, "internal error")
}
