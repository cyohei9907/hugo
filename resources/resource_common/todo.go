package resource_common

// Imaging contains default image processing configuration. This will be fetched
// from site (or language) config.
type Imaging struct {
	// Default image quality setting (1-100). Only used for JPEG images.
	Quality int

	// Resample filter used. See https://github.com/disintegration/imaging
	ResampleFilter string

	// The anchor used in Fill. Default is "smart", i.e. Smart Crop.
	Anchor string
}
