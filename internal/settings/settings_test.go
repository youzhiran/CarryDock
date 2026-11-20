package settings

import (
	"path/filepath"
	"slices"
	"testing"

	"carrydock/internal/storage"
)

func TestGetFontFamilyDefaultAndCustom(t *testing.T) {
	store := storage.NewAtDir(filepath.Join(t.TempDir(), "data"))
	svc := NewService(store)

	if got := svc.GetFontFamily(); got != "" {
		t.Fatalf("expected empty default font family, got %q", got)
	}

	if err := svc.SaveFontFamily("HarmonyOS Sans SC"); err != nil {
		t.Fatalf("SaveFontFamily failed: %v", err)
	}
	if got := svc.GetFontFamily(); got != "HarmonyOS Sans SC" {
		t.Fatalf("expected saved font family, got %q", got)
	}
}

func TestResolveDefaultFontPriority(t *testing.T) {
	cases := []struct {
		name      string
		installed []string
		want      string
	}{
		{
			name:      "pingfang_first",
			installed: []string{"Microsoft YaHei UI", "HarmonyOS Sans SC", "PingFang SC"},
			want:      "PingFang SC",
		},
		{
			name:      "pingfang_generic_second",
			installed: []string{"Microsoft YaHei UI", "HarmonyOS Sans SC", "pingfang"},
			want:      "pingfang",
		},
		{
			name:      "harmony_before_yahei",
			installed: []string{"Microsoft YaHei UI", "harmonyos sans sc"},
			want:      "harmonyos sans sc",
		},
		{
			name:      "harmony_generic_before_yahei",
			installed: []string{"Microsoft YaHei", "HarmonyOS Sans"},
			want:      "HarmonyOS Sans",
		},
		{
			name:      "yahei_ui_fallback",
			installed: []string{"Segoe UI", "Microsoft YaHei", "Microsoft YaHei UI"},
			want:      "Microsoft YaHei UI",
		},
		{
			name:      "none_matched",
			installed: []string{"Arial", "Segoe UI"},
			want:      "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveDefaultFont(tc.installed); got != tc.want {
				t.Fatalf("ResolveDefaultFont(%v) = %q, want %q", tc.installed, got, tc.want)
			}
		})
	}
}

func TestBuildFontStack(t *testing.T) {
	wantDefault := "PingFang SC, PingFang, HarmonyOS Sans SC, HarmonyOS Sans, Microsoft YaHei UI, Microsoft YaHei"
	if got := BuildFontStack(""); got != wantDefault {
		t.Fatalf("BuildFontStack(\"\") = %q, want %q", got, wantDefault)
	}

	wantArial := "Arial, " + wantDefault
	if got := BuildFontStack("Arial"); got != wantArial {
		t.Fatalf("BuildFontStack(\"Arial\") = %q, want %q", got, wantArial)
	}

	// 已在优先级中的字体不会重复出现，且被提升到最前面
	wantHarmony := "HarmonyOS Sans SC, PingFang SC, PingFang, HarmonyOS Sans, Microsoft YaHei UI, Microsoft YaHei"
	if got := BuildFontStack("HarmonyOS Sans SC"); got != wantHarmony {
		t.Fatalf("BuildFontStack(\"HarmonyOS Sans SC\") = %q, want %q", got, wantHarmony)
	}
}

func TestFilterSystemFonts(t *testing.T) {
	raw := []string{
		"",
		"  ",
		"@Microsoft YaHei",
		"@SimSun",
		"Wingdings",
		"wingdings 2",
		"Segoe MDL2 Assets",
		"Segoe Fluent Icons",
		"Segoe UI Emoji",
		"Segoe UI Symbol",
		"Segoe UI Historic",
		"Webdings",
		"Symbol",
		"Marlett",
		"MT Extra",
		"HoloLens MDL2 Assets",
		"JetBrains Mono",
		"jetbrains mono",
		"Arial",
		"PingFang SC",
	}

	got := FilterSystemFonts(raw)
	want := []string{"Arial", "JetBrains Mono", "PingFang SC"}
	if !slices.Equal(got, want) {
		t.Fatalf("FilterSystemFonts = %v, want %v", got, want)
	}
}
