package interfaces

// ApplicationInfo allows configuration of application metadata.
//
// If you want to set non-default values for any of these fields, set the ApplicationInfo field
// in the SDK's Config struct.
type ApplicationInfo struct {
	// ApplicationID is a unique identifier representing the application where the LaunchDarkly SDK is
	// running.
	//
	// This can be specified as any string value as long as it only uses the following characters: ASCII
	// letters, ASCII digits, period, hyphen, underscore. A string containing any other characters will be
	// ignored.
	ApplicationID string

	// ApplicationVersion is a unique identifier representing the version of the application where the
	// LaunchDarkly SDK is running.
	//
	// This can be specified as any string value as long as it only uses the following characters: ASCII
	// letters, ASCII digits, period, hyphen, underscore. A string containing any other characters will be
	// ignored.
	ApplicationVersion string
}
