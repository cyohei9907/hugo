package resources

import (
	"strings"

	"github.com/disintegration/imaging"
)

const (
	defaultJPEGQuality    = 75
	defaultResampleFilter = "box"
)

var (
	imageFormats = map[string]imaging.Format{
		".jpg":  imaging.JPEG,
		".jpeg": imaging.JPEG,
		".png":  imaging.PNG,
		".tif":  imaging.TIFF,
		".tiff": imaging.TIFF,
		".bmp":  imaging.BMP,
		".gif":  imaging.GIF,
	}

	// Add or increment if changes to an image format's processing requires
	// re-generation.
	imageFormatsVersions = map[imaging.Format]int{
		imaging.PNG: 2, // Floyd Steinberg dithering
	}

	// Increment to mark all processed images as stale. Only use when absolutely needed.
	// See the finer grained smartCropVersionNumber and imageFormatsVersions.
	mainImageVersionNumber = 0
)

var anchorPositions = map[string]imaging.Anchor{
	strings.ToLower("Center"):      imaging.Center,
	strings.ToLower("TopLeft"):     imaging.TopLeft,
	strings.ToLower("Top"):         imaging.Top,
	strings.ToLower("TopRight"):    imaging.TopRight,
	strings.ToLower("Left"):        imaging.Left,
	strings.ToLower("Right"):       imaging.Right,
	strings.ToLower("BottomLeft"):  imaging.BottomLeft,
	strings.ToLower("Bottom"):      imaging.Bottom,
	strings.ToLower("BottomRight"): imaging.BottomRight,
}
