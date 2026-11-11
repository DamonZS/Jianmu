// Package all registers every built-in plugin. Only jianmu's main imports
// it: core never imports a plugin, and a plugin never imports another.
package all

import (
	_ "github.com/DamonZS/Jianmu/plugins/budget"
	_ "github.com/DamonZS/Jianmu/plugins/gateway"
	_ "github.com/DamonZS/Jianmu/plugins/groups"
	_ "github.com/DamonZS/Jianmu/plugins/library"
	_ "github.com/DamonZS/Jianmu/plugins/middleware"
	_ "github.com/DamonZS/Jianmu/plugins/otel"
	_ "github.com/DamonZS/Jianmu/plugins/redact"
	_ "github.com/DamonZS/Jianmu/plugins/rotation"
	_ "github.com/DamonZS/Jianmu/plugins/routing"
	_ "github.com/DamonZS/Jianmu/plugins/search"
	_ "github.com/DamonZS/Jianmu/plugins/sessions"
	_ "github.com/DamonZS/Jianmu/plugins/tools"
	_ "github.com/DamonZS/Jianmu/plugins/usage"
	_ "github.com/DamonZS/Jianmu/plugins/warmups"
)
