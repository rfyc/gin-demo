package home

import "context"

func Welcome(ctx context.Context, request *WelcomeRequest) (response *WelcomeResponse, err error) {
	return &WelcomeResponse{Message: "welcome"}, nil
}

type WelcomeRequest struct {
	ID int `json:"id"`
}
type WelcomeResponse struct {
	Message string `json:"message"`
}
