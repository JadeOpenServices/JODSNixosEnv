package discovery

import "github.com/bakanura/gjallarOS/internal/installer/oddc"

// ODDCIdentity converts discovered machine hardware into the structured
// identity consumed by the declarative device collection resolver.
func ODDCIdentity(h Hardware) oddc.Identity {
	return oddc.Identity{
		FormFactor:     h.FormFactor,
		SysVendor:      h.SysVendor,
		ProductName:    h.ProductName,
		ProductVersion: h.ProductVersion,
		BoardVendor:    h.BoardVendor,
		BoardName:      h.BoardName,
		BoardVersion:   h.BoardVersion,
	}
}
