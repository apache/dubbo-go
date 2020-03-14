package identifier

type SubscriberMetadataIdentifier struct {
	revision string
	BaseMetadataIdentifier
}

// getIdentifierKey...
func (mdi *SubscriberMetadataIdentifier) getIdentifierKey(params ...string) string {
	return mdi.BaseMetadataIdentifier.getIdentifierKey(mdi.revision)
}

// getIdentifierKey...
func (mdi *SubscriberMetadataIdentifier) getFilePathKey(params ...string) string {
	return mdi.BaseMetadataIdentifier.getFilePathKey(mdi.revision)
}
