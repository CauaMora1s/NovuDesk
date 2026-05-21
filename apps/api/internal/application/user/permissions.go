package user

import "github.com/novudesk/novudesk/internal/domain/user"

func ApplyOverrides(roleKeys []string, overrides []user.PermissionOverride) []string {
	set := make(map[string]struct{}, len(roleKeys))
	for _, k := range roleKeys {
		set[k] = struct{}{}
	}

	for _, o := range overrides {
		if o.IsGranted {
			set[o.PermissionKey] = struct{}{}
		} else {
			delete(set, o.PermissionKey)
		}
	}

	result := make([]string, 0, len(set))
	for k := range set {
		result = append(result, k)
	}
	return result
}
