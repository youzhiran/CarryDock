package version

import (
	"fmt"
	"strings"
)

// 构建时可通过 -ldflags "-X carrydock/internal/version.Version=... -X carrydock/internal/version.BuildNumber=... -X carrydock/internal/version.BuildTime=..." 注入。
var (
	Version     = "0.9.0"
	BuildNumber = "105"
	BuildTime   = "dev"
)

const (
	WebsiteURL = "https://carrydock.devyi.com"
	GitHubURL  = "https://github.com/youzhiran/CarryDock"
)

// FullVersion 返回带构建号的完整版本号，例如 0.9.0+105。
func FullVersion() string {
	b := strings.TrimSpace(BuildNumber)
	if b == "" {
		return Version
	}
	return fmt.Sprintf("%s+%s", Version, b)
}
