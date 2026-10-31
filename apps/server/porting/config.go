package porting

import (
	"os"
	"strings"
)

// ApplyEnabled enables the operational importer by default. Deployments can
// explicitly disable it with PORTING_APPLY_ENABLED=false. Unknown values disable
// imports as well. The API and CLI/worker must use the same policy.
func ApplyEnabled() bool {
	value := strings.TrimSpace(os.Getenv("PORTING_APPLY_ENABLED"))
	return value == "" || strings.EqualFold(value, "true")
}
