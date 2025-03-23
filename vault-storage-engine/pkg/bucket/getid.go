package bucket

import "github.com/google/uuid"

func GenerateObjectID() string {
	objectID := uuid.New().String()

	return objectID
}
