package packaging

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
)

var appStreamComponentIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// AppStream metainfo files should be named after the component ID, not the
// Debian package name. They are different identifiers for many projects.
func linuxAppStreamFileName(data []byte) (string, error) {
	var component struct {
		XMLName xml.Name `xml:"component"`
		ID      string   `xml:"id"`
	}
	if err := xml.Unmarshal(data, &component); err != nil {
		return "", fmt.Errorf("invalid AppStream metainfo XML: %w", err)
	}
	if component.XMLName.Local != "component" {
		return "", fmt.Errorf("invalid AppStream metainfo root element %q", component.XMLName.Local)
	}
	id := strings.TrimSpace(component.ID)
	if !appStreamComponentIDPattern.MatchString(id) || strings.Contains(id, "..") {
		return "", fmt.Errorf("invalid AppStream component ID %q", id)
	}
	return id + ".metainfo.xml", nil
}
