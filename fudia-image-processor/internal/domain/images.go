package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const MaxImageBytes int64 = 5 * 1024 * 1024

var imageUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func ImageOriginalKey(organizationID, imageID, contentType string) string {
	extension := map[string]string{"image/jpeg": "jpg", "image/png": "png", "image/webp": "webp"}[contentType]
	return fmt.Sprintf("restaurants/%s/products/%s/original.%s", organizationID, imageID, extension)
}
func ImagePrefix(original string) string { return original[:strings.LastIndex(original, "/")+1] }
func ParseImageOriginalKey(key string) (organizationID, imageID string, ok bool) {
	parts := strings.Split(key, "/")
	if len(parts) != 5 || parts[0] != "restaurants" || parts[2] != "products" || !imageUUID.MatchString(parts[1]) || !imageUUID.MatchString(parts[3]) {
		return "", "", false
	}
	if parts[4] != "original.jpg" && parts[4] != "original.png" && parts[4] != "original.webp" {
		return "", "", false
	}
	return parts[1], parts[3], true
}

type ImageRecord struct {
	ID, OrganizationID, LocationID, UploadedBy, OriginalKey, ContentType, Status string
	ProductID                                                                    *string
	SizeBytes                                                                    int64
	Width, Height, Attempts                                                      int
	Generation                                                                   *int64
	ConfirmedAt                                                                  *time.Time
	ExpiresAt                                                                    time.Time
}
