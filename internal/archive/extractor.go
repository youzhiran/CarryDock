package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"carrydock/internal/logger"
	"carrydock/internal/winutil"
	"github.com/ulikunitz/xz"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// Format 表示受支持的归档压缩格式。
type Format int

const (
	FormatUnknown Format = iota
	FormatZip
	FormatTar
	FormatTarGz
	FormatTarBz2
	FormatTarXz
	FormatGz
	FormatBz2
	FormatXz
	FormatRar
	FormatSevenZip
)

// String 返回格式简称。
func (f Format) String() string {
	switch f {
	case FormatZip:
		return "zip"
	case FormatTar:
		return "tar"
	case FormatTarGz:
		return "tar.gz"
	case FormatTarBz2:
		return "tar.bz2"
	case FormatTarXz:
		return "tar.xz"
	case FormatGz:
		return "gz"
	case FormatBz2:
		return "bz2"
	case FormatXz:
		return "xz"
	case FormatRar:
		return "rar"
	case FormatSevenZip:
		return "7z"
	default:
		return "unknown"
	}
}

// PickerExtensions 列出文件选择器支持的归档扩展名。
var PickerExtensions = []string{
	"zip", "tar", "tgz", "tbz", "tbz2", "txz", "gz", "bz2", "xz", "rar", "7z",
}

// DetectFormat 根据文件后缀识别归档格式。
// 修复了原版中 .bz2 与 .xz 虽在选择器白名单但无法识别的问题。
func DetectFormat(path string) Format {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return FormatTarGz
	case strings.HasSuffix(lower, ".tar.bz2"), strings.HasSuffix(lower, ".tbz"), strings.HasSuffix(lower, ".tbz2"):
		return FormatTarBz2
	case strings.HasSuffix(lower, ".tar.xz"), strings.HasSuffix(lower, ".txz"):
		return FormatTarXz
	case strings.HasSuffix(lower, ".tar"):
		return FormatTar
	case strings.HasSuffix(lower, ".zip"):
		return FormatZip
	case strings.HasSuffix(lower, ".gz"):
		return FormatGz
	case strings.HasSuffix(lower, ".bz2"):
		return FormatBz2
	case strings.HasSuffix(lower, ".xz"):
		return FormatXz
	case strings.HasSuffix(lower, ".rar"):
		return FormatRar
	case strings.HasSuffix(lower, ".7z"):
		return FormatSevenZip
	default:
		return FormatUnknown
	}
}

// CompoundExtension 返回完整归档后缀（例如 .tar.gz / .tar.bz2 / .tar.xz / .zip）。
// 修复了原版只取最后一段扩展名导致重命名 .tar.gz 后变为 .gz 的严重 bug。
func CompoundExtension(filename string) string {
	lower := strings.ToLower(filename)
	for _, compound := range []string{".tar.gz", ".tar.bz2", ".tar.xz"} {
		if strings.HasSuffix(lower, compound) {
			return filename[len(filename)-len(compound):]
		}
	}
	return filepath.Ext(filename)
}

// StripArchiveExtension 移除完整归档后缀并返回基本名称。
func StripArchiveExtension(path string) string {
	base := filepath.Base(path)
	ext := CompoundExtension(base)
	if ext == "" {
		return base
	}
	return strings.TrimSuffix(base, ext)
}

// Extract 将归档文件解压到 destination 目录，内置路径穿越防护与中文文件名编码兼容。
func Extract(archivePath, destination string, format Format) error {
	if format == FormatUnknown {
		format = DetectFormat(archivePath)
	}
	if format == FormatUnknown {
		return fmt.Errorf("不支持的归档格式: %s", filepath.Ext(archivePath))
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}

	switch format {
	case FormatZip:
		return extractZip(archivePath, destination)
	case FormatTar, FormatTarGz, FormatTarBz2, FormatTarXz:
		return extractTarBased(archivePath, destination, format)
	case FormatGz, FormatBz2, FormatXz:
		return extractSingleCompressed(archivePath, destination, format)
	case FormatRar:
		return extractWith7z(archivePath, destination, "RAR")
	case FormatSevenZip:
		return extractWith7z(archivePath, destination, "7Z")
	default:
		return fmt.Errorf("不支持的归档格式: %s", format)
	}
}

func extractZip(archivePath, destination string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("打开 ZIP 失败: %w", err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		name := resolveZipEntryName(f)
		if name == "" {
			continue
		}
		targetPath, ok := SafeJoin(destination, name)
		if !ok {
			logger.Warn("检测到可能的路径穿越，已跳过条目: %s", name)
			continue
		}
		if f.FileInfo().IsDir() || strings.HasSuffix(name, "/") {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		if err := copyZipEntry(f, targetPath); err != nil {
			return err
		}
	}
	return nil
}

func copyZipEntry(f *zip.File, targetPath string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, rc)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func extractTarBased(archivePath, destination string, format Format) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var reader io.Reader = f
	switch format {
	case FormatTar:
	case FormatTarGz:
		gz, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("解压 gzip 流失败: %w", err)
		}
		defer gz.Close()
		reader = gz
	case FormatTarBz2:
		reader = bzip2.NewReader(f)
	case FormatTarXz:
		xr, err := xz.NewReader(f)
		if err != nil {
			return fmt.Errorf("解压 xz 流失败: %w", err)
		}
		reader = xr
	}

	tr := tar.NewReader(reader)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取 TAR 条目失败: %w", err)
		}
		name := normalizeArchivePath(hdr.Name)
		if name == "" {
			continue
		}
		targetPath, ok := SafeJoin(destination, name)
		if !ok {
			logger.Warn("检测到可能的路径穿越，已跳过条目: %s", name)
			continue
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		}
	}
	return nil
}

func extractSingleCompressed(archivePath, destination string, format Format) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var reader io.Reader
	switch format {
	case FormatGz:
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gz.Close()
		reader = gz
	case FormatBz2:
		reader = bzip2.NewReader(f)
	case FormatXz:
		xr, err := xz.NewReader(f)
		if err != nil {
			return err
		}
		reader = xr
	}

	outName := StripArchiveExtension(archivePath)
	if outName == "" {
		outName = "extracted"
	}
	targetPath, ok := SafeJoin(destination, outName)
	if !ok {
		return fmt.Errorf("检测到非法输出文件名: %s", outName)
	}
	out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, reader)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

// CreateZipFromDirectory 将整个目录流式压缩为 ZIP 文件（包含顶层目录名，UTF-8 编码）。
func CreateZipFromDirectory(sourceDir, targetZipPath string) error {
	if err := os.MkdirAll(filepath.Dir(targetZipPath), 0o755); err != nil {
		return err
	}
	outFile, err := os.OpenFile(targetZipPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(outFile)

	cleanSrc := filepath.Clean(sourceDir)
	topName := filepath.Base(cleanSrc)

	walkErr := filepath.WalkDir(cleanSrc, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(cleanSrc, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		zipPath := filepath.ToSlash(filepath.Join(topName, rel))
		if d.IsDir() {
			_, err := zw.Create(zipPath + "/")
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		hdr, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		hdr.Name = zipPath
		hdr.Method = zip.Deflate
		hdr.NonUTF8 = false

		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(w, in)
		_ = in.Close()
		return copyErr
	})

	closeZipErr := zw.Close()
	closeFileErr := outFile.Close()
	if walkErr != nil {
		_ = os.Remove(targetZipPath)
		return walkErr
	}
	if closeZipErr != nil {
		_ = os.Remove(targetZipPath)
		return closeZipErr
	}
	if closeFileErr != nil {
		_ = os.Remove(targetZipPath)
		return closeFileErr
	}
	return nil
}

func resolveZipEntryName(f *zip.File) string {
	name := f.Name
	if name == "" {
		return ""
	}
	hasUTF8 := (f.Flags & 0x0800) != 0
	if !hasUTF8 {
		if u := readUnicodeExtraField(f.Extra); u != "" {
			name = u
		} else if !utf8.ValidString(name) {
			if decoded, err := simplifiedchinese.GB18030.NewDecoder().String(name); err == nil && decoded != "" {
				name = decoded
			}
		}
	}
	return normalizeArchivePath(name)
}

func readUnicodeExtraField(extra []byte) string {
	offset := 0
	for offset+4 <= len(extra) {
		headerID := binary.LittleEndian.Uint16(extra[offset : offset+2])
		dataSize := int(binary.LittleEndian.Uint16(extra[offset+2 : offset+4]))
		offset += 4
		if offset+dataSize > len(extra) {
			break
		}
		if headerID == 0x7075 && dataSize >= 5 {
			raw := extra[offset+5 : offset+dataSize]
			if utf8.Valid(raw) {
				return string(raw)
			}
		}
		offset += dataSize
	}
	return ""
}

func normalizeArchivePath(raw string) string {
	s := strings.ReplaceAll(raw, "\\", "/")
	for strings.HasPrefix(s, "./") {
		s = strings.TrimPrefix(s, "./")
	}
	for strings.Contains(s, "//") {
		s = strings.ReplaceAll(s, "//", "/")
	}
	s = strings.TrimLeft(s, "/")
	return strings.TrimSpace(s)
}

// SafeJoin 将归档内的相对路径安全拼接到目标根目录，拒绝任何目录穿越。
func SafeJoin(root, relativePath string) (string, bool) {
	norm := normalizeArchivePath(relativePath)
	if norm == "" {
		return "", false
	}
	absRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return "", false
	}
	joined := filepath.Clean(filepath.Join(absRoot, filepath.FromSlash(norm)))
	absTarget, err := filepath.Abs(joined)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return "", false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", false
	}
	return absTarget, true
}

var common7ZipPaths = []string{
	`C:\Program Files\7-Zip\7z.exe`,
	`C:\Program Files (x86)\7-Zip\7z.exe`,
	`C:\Apps\7-Zip\7z.exe`,
	`D:\Program Files\7-Zip\7z.exe`,
	`D:\Program Files (x86)\7-Zip\7z.exe`,
}

// Find7ZipPath 自动检测本机 7z.exe 路径（常见目录 -> PATH -> Windows 注册表）。
func Find7ZipPath() string {
	for _, p := range common7ZipPaths {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	if p, err := exec.LookPath("7z"); err == nil && p != "" {
		return p
	}
	if p, err := exec.LookPath("7z.exe"); err == nil && p != "" {
		return p
	}
	for _, r := range winutil.Read7ZipRegistryPaths() {
		if r.Exists && r.ExePath != "" {
			return r.ExePath
		}
	}
	return ""
}

// Diagnose7Zip 生成详细的 7-Zip 检测报告，供设置页「检测 7-Zip」展示。
func Diagnose7Zip() (foundPath string, report string) {
	var sb strings.Builder
	sb.WriteString("=== 7z.exe 路径检测开始 ===\n\n")

	sb.WriteString("1. 检查常见安装路径：\n")
	for _, p := range common7ZipPaths {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			sb.WriteString("   ✓ " + p + "\n")
			if foundPath == "" {
				foundPath = p
			}
		} else {
			sb.WriteString("   ✗ " + p + "\n")
		}
	}
	sb.WriteString("\n")

	sb.WriteString("2. 检查系统 PATH 中的 7z 命令：\n")
	if p, err := exec.LookPath("7z"); err == nil && p != "" {
		sb.WriteString("   ✓ " + p + "\n")
		if foundPath == "" {
			foundPath = p
		}
	} else {
		sb.WriteString("   ✗ 未在 PATH 中找到 7z\n")
	}
	sb.WriteString("\n")

	sb.WriteString("3. 检查 Windows 注册表：\n")
	regResults := winutil.Read7ZipRegistryPaths()
	if len(regResults) == 0 {
		sb.WriteString("   - 当前系统无 Windows 注册表\n")
	} else {
		for _, r := range regResults {
			if r.Err != "" {
				sb.WriteString(fmt.Sprintf("   ✗ %s: %s\n", r.Label, r.Err))
				continue
			}
			mark := "✗"
			if r.Exists {
				mark = "✓"
				if foundPath == "" {
					foundPath = r.ExePath
				}
			}
			sb.WriteString(fmt.Sprintf("   %s %s -> %s\n", mark, r.Label, r.ExePath))
		}
	}
	sb.WriteString("\n")

	sb.WriteString("4. 检测结果：\n")
	if foundPath != "" {
		sb.WriteString("   ✓ 成功找到 7z.exe：" + foundPath + "\n")
	} else {
		sb.WriteString("   ✗ 未找到 7z.exe，请安装 7-Zip（https://www.7-zip.org/）\n")
	}
	sb.WriteString("\n=== 检测完成 ===")
	return foundPath, sb.String()
}

func extractWith7z(archivePath, destination, formatLabel string) error {
	sevenZip := Find7ZipPath()
	if sevenZip == "" {
		return fmt.Errorf("无法解压 %s 文件，未找到 7-Zip 安装。请先安装 7-Zip（https://www.7-zip.org/）。", formatLabel)
	}
	logger.Info("使用 7-Zip 解压 (%s): %s -> %s", sevenZip, archivePath, destination)
	cmd := exec.Command(sevenZip, "x", archivePath, "-o"+destination, "-y")
	winutil.HideConsole(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("7-Zip 解压失败: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}
