package home

import "context"

func Hello(ctx context.Context, request HelloRequest) (response HelloResponse, err error) {
	return HelloResponse{Message: "hello"}, nil
}

type HelloRequest struct {
	Name string `json:"name"`
}
type HelloResponse struct {
	Message string `json:"message"`
}
