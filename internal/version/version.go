package version

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// 构建时可通过 -ldflags 或 MYGO_LDFLAGS 注入：
// -X carrydock/internal/version.Version=... -X carrydock/internal/version.BuildNumber=... -X 'carrydock/internal/version.BuildTime=...'
var (
	Version     = "0.9.1"
	BuildNumber = "106"
	BuildTime   = "dev"
)

const (
	WebsiteURL = "https://carrydock.devyi.com"
	GitHubURL  = "https://github.com/youzhiran/CarryDock"
)

const buildTimeLayout = "2006-01-02 15:04:05"

var (
	resolvedBuildTimeOnce sync.Once
	resolvedBuildTimeVal  string
)

// FullVersion 返回带构建号的完整版本号，例如 0.9.0+105。
func FullVersion() string {
	b := strings.TrimSpace(BuildNumber)
	if b == "" {
		return Version
	}
	return fmt.Sprintf("%s+%s", Version, b)
}

// DisplayBuildTime 返回用于界面展示的构建时间。
// 优先使用编译期注入的 BuildTime；若仍为默认值 "dev" 或空字符串，
// 则回退到 Go 构建信息中的 VCS 提交时间或当前可执行文件的修改时间。
func DisplayBuildTime() string {
	if s := strings.TrimSpace(BuildTime); s != "" && !strings.EqualFold(s, "dev") {
		return s
	}
	resolvedBuildTimeOnce.Do(func() {
		resolvedBuildTimeVal = resolveBuildTime(BuildTime, readVCSBuildTime, readExeModTime)
	})
	return resolvedBuildTimeVal
}

func resolveBuildTime(injected string, readVCS func() (time.Time, bool, bool), readExeMod func() (time.Time, bool)) string {
	if s := strings.TrimSpace(injected); s != "" && !strings.EqualFold(s, "dev") {
		return s
	}
	var vcsTime time.Time
	var vcsModified, vcsOK bool
	if readVCS != nil {
		vcsTime, vcsModified, vcsOK = readVCS()
	}
	if vcsOK && !vcsModified {
		return vcsTime.In(time.Local).Format(buildTimeLayout)
	}
	if readExeMod != nil {
		if exeTime, ok := readExeMod(); ok {
			return exeTime.In(time.Local).Format(buildTimeLayout)
		}
	}
	if vcsOK {
		return vcsTime.In(time.Local).Format(buildTimeLayout)
	}
	return "dev"
}

func readVCSBuildTime() (time.Time, bool, bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return time.Time{}, false, false
	}
	var parsed time.Time
	var hasTime bool
	var modified bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.time":
			if t, err := time.Parse(time.RFC3339, strings.TrimSpace(s.Value)); err == nil {
				parsed = t
				hasTime = true
			}
		case "vcs.modified":
			modified = strings.EqualFold(strings.TrimSpace(s.Value), "true")
		}
	}
	return parsed, modified, hasTime
}

func readExeModTime() (time.Time, bool) {
	exe, err := os.Executable()
	if err != nil {
		return time.Time{}, false
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	st, err := os.Stat(exe)
	if err != nil || st.IsDir() {
		return time.Time{}, false
	}
	return st.ModTime(), true
}
