package installercheck

import "encoding/json"

func matchesPresetWebApplicationList(raw json.RawMessage) bool {
	var applications []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &applications); err != nil ||
		applications == nil {
		return false
	}

	for _, application := range applications {
		if application == nil {
			return false
		}

		for key := range application {
			if key != "id" && key != "endpoint" {
				return false
			}
		}

		rawID, ok := application["id"]
		if !ok {
			return false
		}

		var id string
		if err := json.Unmarshal(rawID, &id); err != nil {
			return false
		}

		if rawEndpoint, ok := application["endpoint"]; ok {
			var endpoint string
			if err := json.Unmarshal(rawEndpoint, &endpoint); err != nil {
				return false
			}
		}
	}

	return true
}
