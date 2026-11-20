package archive

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectFormatAndCompoundExtensions(t *testing.T) {
	cases := []struct {
		name     string
		wantFmt  Format
		wantExt  string
		wantBase string
	}{
		{"tool.zip", FormatZip, ".zip", "tool"},
		{"tool.tar.gz", FormatTarGz, ".tar.gz", "tool"},
		{"tool.tgz", FormatTarGz, ".tgz", "tool"},
		{"tool.tar.bz2", FormatTarBz2, ".tar.bz2", "tool"},
		{"tool.tar.xz", FormatTarXz, ".tar.xz", "tool"},
		{"tool.tar", FormatTar, ".tar", "tool"},
		{"tool.gz", FormatGz, ".gz", "tool"},
		{"tool.bz2", FormatBz2, ".bz2", "tool"},
		{"tool.xz", FormatXz, ".xz", "tool"},
		{"tool.rar", FormatRar, ".rar", "tool"},
		{"tool.7z", FormatSevenZip, ".7z", "tool"},
		{"readme.txt", FormatUnknown, ".txt", "readme"},
	}

	for _, tc := range cases {
		if got := DetectFormat(tc.name); got != tc.wantFmt {
			t.Errorf("DetectFormat(%q) = %v, want %v", tc.name, got, tc.wantFmt)
		}
		if got := CompoundExtension(tc.name); got != tc.wantExt {
			t.Errorf("CompoundExtension(%q) = %q, want %q", tc.name, got, tc.wantExt)
		}
		if got := StripArchiveExtension(tc.name); got != tc.wantBase {
			t.Errorf("StripArchiveExtension(%q) = %q, want %q", tc.name, got, tc.wantBase)
		}
	}
}

func TestSafeJoinPreventsZipSlip(t *testing.T) {
	root := t.TempDir()

	if _, ok := SafeJoin(root, "../evil.exe"); ok {
		t.Fatal("expected ../evil.exe to be rejected")
	}
	if _, ok := SafeJoin(root, "a/../../evil.exe"); ok {
		t.Fatal("expected a/../../evil.exe to be rejected")
	}
	p, ok := SafeJoin(root, "./sub/app.exe")
	if !ok {
		t.Fatal("expected ./sub/app.exe to be accepted")
	}
	if want := filepath.Join(root, "sub", "app.exe"); p != want {
		t.Fatalf("SafeJoin got %q, want %q", p, want)
	}
}

func TestZipCreateAndExtractRoundtrip(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "示例软件")
	if err := os.MkdirAll(filepath.Join(srcDir, "子目录"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "子目录", "hello.txt"), []byte("carrydock"), 0o644); err != nil {
		t.Fatal(err)
	}

	zipPath := filepath.Join(tmp, "out.zip")
	if err := CreateZipFromDirectory(srcDir, zipPath); err != nil {
		t.Fatalf("CreateZipFromDirectory failed: %v", err)
	}

	destDir := filepath.Join(tmp, "extracted")
	if err := Extract(zipPath, destDir, FormatZip); err != nil {
		t.Fatalf("Extract zip failed: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(destDir, "示例软件", "子目录", "hello.txt"))
	if err != nil {
		t.Fatalf("expected extracted file: %v", err)
	}
	if string(got) != "carrydock" {
		t.Fatalf("content = %q, want carrydock", string(got))
	}
}

func TestExtractTarGz(t *testing.T) {
	tmp := t.TempDir()
	tgzPath := filepath.Join(tmp, "demo.tar.gz")

	f, err := os.Create(tgzPath)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	payload := []byte("echo hi")
	if err := tw.WriteHeader(&tar.Header{Name: "demo/run.bat", Mode: 0o644, Size: int64(len(payload))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(payload); err != nil {
		t.Fatal(err)
	}
	_ = tw.Close()
	_ = gw.Close()
	_ = f.Close()

	dest := filepath.Join(tmp, "out")
	if err := Extract(tgzPath, dest, FormatTarGz); err != nil {
		t.Fatalf("Extract tar.gz failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "demo", "run.bat")); err != nil {
		t.Fatalf("expected extracted file demo/run.bat: %v", err)
	}
}
