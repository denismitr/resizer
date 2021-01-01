package backoffice

type errorResponse struct {
	Message string `json:"message"`
	Details interface{}
}

func internalError(err error) (int, errorResponse) {
	return 500, errorResponse{Message: err.Error()}
}

func badRequest(err error) (int, errorResponse) {
	return 400, errorResponse{Message: err.Error()}
}

func unprocessableEntity(err error) (int, errorResponse) {
	return 422, errorResponse{Message: err.Error()}
}
