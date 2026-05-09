package user

import "jota/server/internal/kv"

type Metadata struct {
	Spotify string `json:"spotify,omitempty"`
}

var userMetadataBucket = kv.UseBucket("user-metadata")

func (s *UserService) GetUserMetadata(name string) (Metadata, error) {
	var metadata Metadata
	metadataError := userMetadataBucket.GetObject(name, &metadata)
	if metadataError != nil {
		return Metadata{}, metadataError
	}

	return metadata, nil
}

func (s *UserService) SetUserMetadata(name string, metadata Metadata) error {
	return userMetadataBucket.SetObject(name, metadata)
}
