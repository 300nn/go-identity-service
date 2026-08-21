package eventcodec

import (
	usereventv1 "CrudTutorialProject/internal/gen/events/user/v1"
	"fmt"

	"google.golang.org/protobuf/proto"
)

const (
	ContentTypeProtobuf = "application/x-protobuf"

	ProtoMessageUserRegistered = "events.user.v1.UserRegistered"
	EventVersionV1             = "v1"
)

func MarshalUserRegistered(userID int64, email string, role string) ([]byte, error) {
	payload := &usereventv1.UserRegistered{
		UserId: userID,
		Email:  email,
		Role:   role,
	}

	data, err := proto.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal user registered protobuf: %w", err)
	}

	return data, nil
}

func UnmarshalUserRegistered(data []byte) (*usereventv1.UserRegistered, error) {
	var payload usereventv1.UserRegistered

	if err := proto.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal user registered protobuf: %w", err)
	}

	return &payload, nil
}
