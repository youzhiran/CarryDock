package version

import (
	"testing"
	"time"
)

func TestResolveBuildTime(t *testing.T) {
	vcsT := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	exeT := time.Date(2026, 11, 20, 18, 30, 45, 0, time.Local)

	// 1. 编译期注入了合法构建时间时直接返回注入值
	if got := resolveBuildTime("2026-11-20 17:45:27", nil, nil); got != "2026-11-20 17:45:27" {
		t.Fatalf("injected build time mismatch: got %q", got)
	}

	// 2. 默认 "dev" 且工作区干净时回退到 VCS 提交时间
	gotClean := resolveBuildTime("dev",
		func() (time.Time, bool, bool) { return vcsT, false, true },
		func() (time.Time, bool) { return exeT, true },
	)
	wantClean := vcsT.In(time.Local).Format(buildTimeLayout)
	if gotClean != wantClean {
		t.Fatalf("clean VCS fallback mismatch: got %q, want %q", gotClean, wantClean)
	}

	// 3. 空值且工作区存在未提交修改时优先使用可执行文件修改时间
	gotDirty := resolveBuildTime("",
		func() (time.Time, bool, bool) { return vcsT, true, true },
		func() (time.Time, bool) { return exeT, true },
	)
	wantDirty := exeT.In(time.Local).Format(buildTimeLayout)
	if gotDirty != wantDirty {
		t.Fatalf("dirty workspace exe fallback mismatch: got %q, want %q", gotDirty, wantDirty)
	}

	// 4. 脏工作区但无法读取 exe 时间时仍回退 VCS 时间
	gotVCSBackup := resolveBuildTime("DEV",
		func() (time.Time, bool, bool) { return vcsT, true, true },
		func() (time.Time, bool) { return time.Time{}, false },
	)
	if gotVCSBackup != wantClean {
		t.Fatalf("VCS secondary fallback mismatch: got %q, want %q", gotVCSBackup, wantClean)
	}

	// 5. 全部不可用时兜底返回 "dev"
	gotNone := resolveBuildTime("dev",
		func() (time.Time, bool, bool) { return time.Time{}, false, false },
		func() (time.Time, bool) { return time.Time{}, false },
	)
	if gotNone != "dev" {
		t.Fatalf("final fallback mismatch: got %q, want %q", gotNone, "dev")
	}
}
