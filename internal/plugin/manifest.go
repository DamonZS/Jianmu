package plugin

import "encoding/json"

// metadataBlock is a package.json's host metadata: its jianmu block, or
// the magpie block a plugin written before jianmu renamed it keeps. The
// field names are the same, so a reader decodes whichever it finds; one
// that says both names jianmu's, keeping the old block for older readers.
// nil for neither.
func metadataBlock(b []byte) json.RawMessage {
	var pkg struct {
		Jianmu json.RawMessage `json:"jianmu"`
		Magpie json.RawMessage `json:"magpie"`
	}
	if json.Unmarshal(b, &pkg) != nil {
		return nil
	}
	if len(pkg.Jianmu) > 0 && string(pkg.Jianmu) != "null" {
		return pkg.Jianmu
	}
	if len(pkg.Magpie) > 0 && string(pkg.Magpie) != "null" {
		return pkg.Magpie
	}
	return nil
}
