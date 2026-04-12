package databases

import (
	"fmt"
	"strings"
)

// EnforceStorageRLS возвращает ошибку, если rollout требует effective RLS, но текущий status ему не соответствует.
func EnforceStorageRLS(status StorageRLSStatus, required bool) error {
	if !required || status.Effective() {
		return nil
	}
	warnings := status.Warnings()
	if len(warnings) == 0 {
		return fmt.Errorf("storage RLS is required but not effective for current DB role")
	}
	return fmt.Errorf("storage RLS is required but not effective: %s", strings.Join(warnings, "; "))
}
