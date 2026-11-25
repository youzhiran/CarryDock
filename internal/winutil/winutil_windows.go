//go:build windows

package winutil

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	modVersion = windows.NewLazySystemDLL("version.dll")
	modShell32 = windows.NewLazySystemDLL("shell32.dll")
	modUser32  = windows.NewLazySystemDLL("user32.dll")
	modGdi32   = windows.NewLazySystemDLL("gdi32.dll")
	modOle32   = windows.NewLazySystemDLL("ole32.dll")

	procGetFileVersionInfoSizeW = modVersion.NewProc("GetFileVersionInfoSizeW")
	procGetFileVersionInfoW     = modVersion.NewProc("GetFileVersionInfoW")
	procVerQueryValueW          = modVersion.NewProc("VerQueryValueW")

	procExtractIconExW             = modShell32.NewProc("ExtractIconExW")
	procSHGetFileInfoW             = modShell32.NewProc("SHGetFileInfoW")
	procShellExecuteW              = modShell32.NewProc("ShellExecuteW")
	procILCreateFromPathW          = modShell32.NewProc("ILCreateFromPathW")
	procILFree                     = modShell32.NewProc("ILFree")
	procSHOpenFolderAndSelectItems = modShell32.NewProc("SHOpenFolderAndSelectItems")

	procCoInitializeEx = modOle32.NewProc("CoInitializeEx")
	procCoUninitialize = modOle32.NewProc("CoUninitialize")

	procPrivateExtractIconsW = modUser32.NewProc("PrivateExtractIconsW")
	procGetIconInfo          = modUser32.NewProc("GetIconInfo")
	procDestroyIcon          = modUser32.NewProc("DestroyIcon")
	procGetDC                = modUser32.NewProc("GetDC")
	procReleaseDC            = modUser32.NewProc("ReleaseDC")

	procGetObjectW   = modGdi32.NewProc("GetObjectW")
	procGetDIBits    = modGdi32.NewProc("GetDIBits")
	procDeleteObject = modGdi32.NewProc("DeleteObject")
)

const (
	biRGB                   = 0
	dibRGBColors            = 0
	swShowNormal            = 1
	shgfiIcon               = 0x000000100
	shgfiLargeIcon          = 0x000000000
	coinitApartmentThreaded = 0x2
	sOK                     = 0x00000000
	sFalse                  = 0x00000001
	rpcEChangedMode         = 0x80010106

	vsFixedFileInfoSignature = 0xFEEF04BD
)

// FileMetadata 保存从 Windows 可执行文件版本资源中提取的描述、产品信息与原生版本号。
type FileMetadata struct {
	Description      string
	ProductName      string
	OriginalFilename string
	Version          string
}

type vsFixedFileInfo struct {
	Signature        uint32
	StrucVersion     uint32
	FileVersionMS    uint32
	FileVersionLS    uint32
	ProductVersionMS uint32
	ProductVersionLS uint32
	FileFlagsMask    uint32
	FileFlags        uint32
	FileOS           uint32
	FileType         uint32
	FileSubtype      uint32
	FileDateMS       uint32
	FileDateLS       uint32
}

type shFileInfoW struct {
	HIcon         uintptr
	IIcon         int32
	DwAttributes  uint32
	SzDisplayName [260]uint16
	SzTypeName    [80]uint16
}

type iconInfo struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
}

type bitmap struct {
	BmType       int32
	BmWidth      int32
	BmHeight     int32
	BmWidthBytes int32
	BmPlanes     uint16
	BmBitsPixel  uint16
	BmBits       uintptr
}

type bitmapInfoHeader struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type bitmapInfo struct {
	BmiHeader bitmapInfoHeader
	BmiColors [1]uint32
}

// SamePath 以 Windows 不区分大小写的方式比较两个路径是否指向同一位置。
func SamePath(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return a == b
	}
	ca := filepath.Clean(a)
	cb := filepath.Clean(b)
	if absA, err := filepath.Abs(ca); err == nil {
		ca = absA
	}
	if absB, err := filepath.Abs(cb); err == nil {
		cb = absB
	}
	return strings.EqualFold(ca, cb)
}

// WithFileLock 在指定的锁文件上获取跨进程独占文件锁并执行 fn。
func WithFileLock(lockPath string, fn func() error) error {
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	h := windows.Handle(f.Fd())
	var ol windows.Overlapped
	if err := windows.LockFileEx(h, windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, &ol); err != nil {
		return err
	}
	defer func() {
		_ = windows.UnlockFileEx(h, 0, 1, 0, &ol)
	}()

	return fn()
}

// GetFileMetadata 从 Windows 可执行文件的版本资源中一次性提取 FileDescription 与原生版本号。
func GetFileMetadata(exePath string) FileMetadata {
	pathPtr, err := windows.UTF16PtrFromString(exePath)
	if err != nil {
		return FileMetadata{}
	}
	var handle uint32
	size, _, _ := procGetFileVersionInfoSizeW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&handle)),
	)
	if size == 0 {
		return FileMetadata{}
	}

	buf := make([]byte, size)
	ok, _, _ := procGetFileVersionInfoW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		size,
		uintptr(unsafe.Pointer(&buf[0])),
	)
	if ok == 0 {
		return FileMetadata{}
	}

	// 收集并去重语言代码页组合：优先 \VarFileInfo\Translation，其次常见代码页回退
	langCodepages := make([]string, 0, 8)
	seen := make(map[string]bool, 8)
	addLangCodepage := func(key string) {
		if key != "" && !seen[key] {
			seen[key] = true
			langCodepages = append(langCodepages, key)
		}
	}

	transKey, _ := windows.UTF16PtrFromString(`\VarFileInfo\Translation`)
	var transPtr unsafe.Pointer
	var transLen uint32
	r, _, _ := procVerQueryValueW.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(transKey)),
		uintptr(unsafe.Pointer(&transPtr)),
		uintptr(unsafe.Pointer(&transLen)),
	)
	if r != 0 && transPtr != nil && transLen >= 4 {
		raw := unsafe.Slice((*byte)(transPtr), int(transLen))
		for offset := 0; offset+4 <= len(raw); offset += 4 {
			lang := binary.LittleEndian.Uint16(raw[offset : offset+2])
			cp := binary.LittleEndian.Uint16(raw[offset+2 : offset+4])
			addLangCodepage(fmt.Sprintf("%04x%04x", lang, cp))
		}
	}
	for _, fallback := range []string{"080404b0", "040904b0", "040904e4", "000004b0"} {
		addLangCodepage(fallback)
	}

	queryString := func(langCodepage, field string) string {
		q, err := windows.UTF16PtrFromString(`\StringFileInfo\` + langCodepage + `\` + field)
		if err != nil {
			return ""
		}
		var valPtr unsafe.Pointer
		var valLen uint32
		r, _, _ := procVerQueryValueW.Call(
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(unsafe.Pointer(q)),
			uintptr(unsafe.Pointer(&valPtr)),
			uintptr(unsafe.Pointer(&valLen)),
		)
		if r == 0 || valPtr == nil || valLen == 0 {
			return ""
		}
		u16 := unsafe.Slice((*uint16)(valPtr), int(valLen))
		return strings.TrimSpace(windows.UTF16ToString(u16))
	}

	var meta FileMetadata
	for _, key := range langCodepages {
		if desc := queryString(key, "FileDescription"); desc != "" {
			meta.Description = desc
			break
		}
	}
	for _, key := range langCodepages {
		if prod := queryString(key, "ProductName"); prod != "" {
			meta.ProductName = prod
			break
		}
	}
	for _, key := range langCodepages {
		if orig := queryString(key, "OriginalFilename"); orig != "" {
			meta.OriginalFilename = orig
			break
		}
	}

	// 版本号按 ProductVersion -> FileVersion -> VS_FIXEDFILEINFO 优先级提取
	for _, field := range []string{"ProductVersion", "FileVersion"} {
		for _, key := range langCodepages {
			if ver := normalizeVersionString(queryString(key, field)); ver != "" {
				meta.Version = ver
				break
			}
		}
		if meta.Version != "" {
			break
		}
	}

	if meta.Version == "" {
		rootKey, _ := windows.UTF16PtrFromString(`\`)
		var fixedPtr unsafe.Pointer
		var fixedLen uint32
		rFixed, _, _ := procVerQueryValueW.Call(
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(unsafe.Pointer(rootKey)),
			uintptr(unsafe.Pointer(&fixedPtr)),
			uintptr(unsafe.Pointer(&fixedLen)),
		)
		if rFixed != 0 && fixedPtr != nil && fixedLen >= uint32(unsafe.Sizeof(vsFixedFileInfo{})) {
			fixed := (*vsFixedFileInfo)(fixedPtr)
			if fixed.Signature == vsFixedFileInfoSignature {
				if ver := formatFixedVersion(fixed.ProductVersionMS, fixed.ProductVersionLS); ver != "" {
					meta.Version = ver
				} else if ver := formatFixedVersion(fixed.FileVersionMS, fixed.FileVersionLS); ver != "" {
					meta.Version = ver
				}
			}
		}
	}

	return meta
}

// GetFileDescription 从 Windows 可执行文件的版本资源中提取 FileDescription。
func GetFileDescription(exePath string) string {
	return GetFileMetadata(exePath).Description
}

// GetFileVersion 从 Windows 可执行文件的版本资源中提取原生版本号。
func GetFileVersion(exePath string) string {
	return GetFileMetadata(exePath).Version
}

func formatFixedVersion(ms, ls uint32) string {
	major := ms >> 16
	minor := ms & 0xFFFF
	build := ls >> 16
	rev := ls & 0xFFFF
	if major == 0 && minor == 0 && build == 0 && rev == 0 {
		return ""
	}
	if rev != 0 {
		return fmt.Sprintf("%d.%d.%d.%d", major, minor, build, rev)
	}
	if build != 0 {
		return fmt.Sprintf("%d.%d.%d", major, minor, build)
	}
	return fmt.Sprintf("%d.%d", major, minor)
}

func normalizeVersionString(raw string) string {
	s := strings.TrimSpace(strings.ReplaceAll(raw, "\x00", ""))
	if s == "" {
		return ""
	}
	switch strings.ToLower(s) {
	case "n/a", "none", "unknown", "null":
		return ""
	}

	hasDigit := false
	onlyCommaDotDigits := true
	hasComma := false
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b >= '0' && b <= '9' {
			hasDigit = true
			continue
		}
		if b == ',' {
			hasComma = true
			continue
		}
		if b == '.' || b == ' ' || b == '\t' {
			continue
		}
		onlyCommaDotDigits = false
	}
	if !hasDigit {
		return ""
	}

	if len(s) >= 2 && (s[0] == 'v' || s[0] == 'V') && s[1] >= '0' && s[1] <= '9' {
		s = s[1:]
	}

	if hasComma && onlyCommaDotDigits {
		s = strings.ReplaceAll(s, ",", ".")
	}

	// 清理点号两侧的多余空白（如 "1 . 2 . 3" 或逗号替换后残留空格）
	if strings.Contains(s, ".") {
		parts := strings.Split(s, ".")
		for i, part := range parts {
			parts[i] = strings.TrimSpace(part)
		}
		s = strings.Join(parts, ".")
	}

	if s == "" {
		return ""
	}

	// 过滤全零版本（如 "0"、"0.0"、"0.0.0.0"）
	allZeroParts := true
	for _, part := range strings.Split(s, ".") {
		if part == "" {
			allZeroParts = false
			break
		}
		for i := 0; i < len(part); i++ {
			if part[i] != '0' {
				allZeroParts = false
				break
			}
		}
		if !allZeroParts {
			break
		}
	}
	if allZeroParts {
		return ""
	}

	return s
}

// ExtractIconPNG 从 exe/ico/lnk/bat 等文件提取指定目标尺寸的主图标并编码为 PNG。
// 优先使用 PrivateExtractIconsW 直接按目标尺寸提取 PE 内嵌高清图标（无资源管理器边框和角落缩放留白），
// 依次回退到 ExtractIconExW 与 SHGetFileInfoW；并通过 GetDIBits 正确处理 32 位透明通道与旧式掩码图标。
func ExtractIconPNG(exePath string, preferredSize int) ([]byte, error) {
	if preferredSize <= 0 {
		preferredSize = 64
	}
	cleanPath := filepath.Clean(exePath)
	pathPtr, err := windows.UTF16PtrFromString(cleanPath)
	if err != nil {
		return nil, err
	}

	hIcon := extractHIcon(pathPtr, preferredSize)
	if hIcon == 0 {
		return nil, fmt.Errorf("未找到有效图标句柄")
	}
	defer procDestroyIcon.Call(hIcon)

	img, err := hiconToNRGBA(hIcon)
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func extractHIcon(pathPtr *uint16, preferredSize int) uintptr {
	// 1. 优先使用 PrivateExtractIconsW 获取指定尺寸的高清原生图标
	sizes := []int{preferredSize, 64, 48, 32}
	for _, sz := range sizes {
		if sz <= 0 {
			continue
		}
		var hIcon uintptr
		var iconID uint32
		r, _, _ := procPrivateExtractIconsW.Call(
			uintptr(unsafe.Pointer(pathPtr)),
			0,
			uintptr(sz),
			uintptr(sz),
			uintptr(unsafe.Pointer(&hIcon)),
			uintptr(unsafe.Pointer(&iconID)),
			1,
			0,
		)
		if r > 0 && r != 0xFFFFFFFF && hIcon != 0 {
			return hIcon
		}
	}

	// 2. 回退到 ExtractIconExW
	var largeIcon, smallIcon uintptr
	count, _, _ := procExtractIconExW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		uintptr(unsafe.Pointer(&largeIcon)),
		uintptr(unsafe.Pointer(&smallIcon)),
		1,
	)
	if count > 0 {
		if largeIcon != 0 {
			if smallIcon != 0 {
				procDestroyIcon.Call(smallIcon)
			}
			return largeIcon
		}
		if smallIcon != 0 {
			return smallIcon
		}
	}

	// 3. 对 .bat/.cmd/.lnk 等无内嵌图标的文件，回退到 SHGetFileInfoW 获取系统关联图标
	var sfi shFileInfoW
	r, _, _ := procSHGetFileInfoW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		uintptr(unsafe.Pointer(&sfi)),
		unsafe.Sizeof(sfi),
		shgfiIcon|shgfiLargeIcon,
	)
	if r != 0 && sfi.HIcon != 0 {
		return sfi.HIcon
	}
	return 0
}

func hiconToNRGBA(hIcon uintptr) (*image.NRGBA, error) {
	var ii iconInfo
	r, _, _ := procGetIconInfo.Call(hIcon, uintptr(unsafe.Pointer(&ii)))
	if r == 0 {
		return nil, fmt.Errorf("GetIconInfo 失败")
	}
	defer func() {
		if ii.HbmColor != 0 {
			procDeleteObject.Call(ii.HbmColor)
		}
		if ii.HbmMask != 0 {
			procDeleteObject.Call(ii.HbmMask)
		}
	}()

	hdcScreen, _, _ := procGetDC.Call(0)
	if hdcScreen == 0 {
		return nil, fmt.Errorf("GetDC 失败")
	}
	defer procReleaseDC.Call(0, hdcScreen)

	// 情况 1：单色图标（无 HbmColor，仅 HbmMask 上下半区分别存放 AND/XOR 掩码）
	if ii.HbmColor == 0 {
		if ii.HbmMask == 0 {
			return nil, fmt.Errorf("图标位图为空")
		}
		var bmp bitmap
		if r, _, _ := procGetObjectW.Call(ii.HbmMask, unsafe.Sizeof(bmp), uintptr(unsafe.Pointer(&bmp))); r == 0 {
			return nil, fmt.Errorf("GetObjectW 失败")
		}
		w := int(bmp.BmWidth)
		fullH := int(bmp.BmHeight)
		h := fullH / 2
		if w <= 0 || h <= 0 || w > 1024 || h > 1024 {
			return nil, fmt.Errorf("无效的单色图标尺寸: %dx%d", w, h)
		}
		maskBits, err := readBitmapBGRA(hdcScreen, ii.HbmMask, w, fullH)
		if err != nil {
			return nil, err
		}
		rgba := image.NewNRGBA(image.Rect(0, 0, w, h))
		halfBytes := w * h * 4
		for i := 0; i < halfBytes; i += 4 {
			andWhite := maskBits[i] != 0 || maskBits[i+1] != 0 || maskBits[i+2] != 0
			xorWhite := maskBits[halfBytes+i] != 0 || maskBits[halfBytes+i+1] != 0 || maskBits[halfBytes+i+2] != 0
			switch {
			case !andWhite && !xorWhite: // 不透明黑
				rgba.Pix[i+3] = 255
			case !andWhite && xorWhite: // 不透明白
				rgba.Pix[i+0], rgba.Pix[i+1], rgba.Pix[i+2], rgba.Pix[i+3] = 255, 255, 255, 255
			case andWhite && xorWhite: // 反色像素：以深灰可见显示
				rgba.Pix[i+0], rgba.Pix[i+1], rgba.Pix[i+2], rgba.Pix[i+3] = 64, 64, 64, 255
			}
		}
		return rgba, nil
	}

	// 情况 2：彩色图标（32 位带 Alpha 或 24/8/4 位依赖 HbmMask 的旧式图标）
	var bmp bitmap
	if r, _, _ := procGetObjectW.Call(ii.HbmColor, unsafe.Sizeof(bmp), uintptr(unsafe.Pointer(&bmp))); r == 0 {
		return nil, fmt.Errorf("GetObjectW 失败")
	}
	w := int(bmp.BmWidth)
	h := int(bmp.BmHeight)
	if w <= 0 || h <= 0 || w > 1024 || h > 1024 {
		return nil, fmt.Errorf("无效的图标尺寸: %dx%d", w, h)
	}

	colorBits, err := readBitmapBGRA(hdcScreen, ii.HbmColor, w, h)
	if err != nil {
		return nil, err
	}

	hasAlpha := false
	for i := 3; i < len(colorBits); i += 4 {
		if colorBits[i] != 0 {
			hasAlpha = true
			break
		}
	}

	var maskBits []byte
	if !hasAlpha && ii.HbmMask != 0 {
		maskBits, _ = readBitmapBGRA(hdcScreen, ii.HbmMask, w, h)
	}

	rgba := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(colorBits); i += 4 {
		b, g, rCh, a := colorBits[i], colorBits[i+1], colorBits[i+2], colorBits[i+3]
		if !hasAlpha {
			if len(maskBits) == len(colorBits) {
				// AND 掩码为白色（非 0）处表示透明，为黑色（0）处表示不透明
				if maskBits[i] != 0 || maskBits[i+1] != 0 || maskBits[i+2] != 0 {
					rCh, g, b, a = 0, 0, 0, 0
				} else {
					a = 255
				}
			} else {
				a = 255
			}
		}
		rgba.Pix[i+0] = rCh
		rgba.Pix[i+1] = g
		rgba.Pix[i+2] = b
		rgba.Pix[i+3] = a
	}
	return rgba, nil
}

func readBitmapBGRA(hdc, hbm uintptr, width, height int) ([]byte, error) {
	var bi bitmapInfo
	bi.BmiHeader.BiSize = uint32(unsafe.Sizeof(bi.BmiHeader))
	bi.BmiHeader.BiWidth = int32(width)
	bi.BmiHeader.BiHeight = -int32(height) // top-down DIB
	bi.BmiHeader.BiPlanes = 1
	bi.BmiHeader.BiBitCount = 32
	bi.BmiHeader.BiCompression = biRGB

	buf := make([]byte, width*height*4)
	lines, _, _ := procGetDIBits.Call(
		hdc,
		hbm,
		0,
		uintptr(height),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&bi)),
		dibRGBColors,
	)
	if lines == 0 {
		return nil, fmt.Errorf("GetDIBits 读取位图像素失败")
	}
	return buf, nil
}

// LaunchExecutable 使用 Win32 ShellExecuteW 启动可执行文件。
// 相比 cmd.exe /c start，能正确处理空格、特殊字符、UAC 提权请求并避免控制台闪现。
func LaunchExecutable(exePath, fallbackWorkDir string) error {
	workDir := filepath.Dir(exePath)
	if workDir == "" || workDir == "." {
		workDir = fallbackWorkDir
	}
	return shellExecuteW("open", exePath, "", workDir)
}

func shellExecuteW(verb, file, params, workDir string) error {
	var verbPtr *uint16
	if verb != "" {
		var err error
		verbPtr, err = windows.UTF16PtrFromString(verb)
		if err != nil {
			return err
		}
	}
	filePtr, err := windows.UTF16PtrFromString(file)
	if err != nil {
		return err
	}
	var paramsPtr *uint16
	if params != "" {
		paramsPtr, err = windows.UTF16PtrFromString(params)
		if err != nil {
			return err
		}
	}
	var dirPtr *uint16
	if workDir != "" {
		dirPtr, err = windows.UTF16PtrFromString(workDir)
		if err != nil {
			return err
		}
	}
	ret, _, _ := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(filePtr)),
		uintptr(unsafe.Pointer(paramsPtr)),
		uintptr(unsafe.Pointer(dirPtr)),
		swShowNormal,
	)
	// ShellExecuteW 返回值 <= 32 表示错误码
	if ret <= 32 {
		return fmt.Errorf("ShellExecuteW 启动失败 (错误码 %d)", ret)
	}
	return nil
}

// HideConsole 隐藏 Windows 下子进程弹出的黑色控制台窗口。
func HideConsole(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

func buildBatchCmdLine(cmdExe, scriptPath string) string {
	clean := filepath.Clean(strings.TrimSpace(scriptPath))
	return fmt.Sprintf(`"%s" /c ""%s""`, cmdExe, clean)
}

// LaunchBatchScript 在独立新控制台窗口中异步启动批处理脚本，避免路径含空格或特殊字符时 cmd /c start 解析失败。
func LaunchBatchScript(scriptPath string) error {
	trimmed := strings.TrimSpace(scriptPath)
	if trimmed == "" {
		return fmt.Errorf("批处理脚本路径为空")
	}
	clean := filepath.Clean(trimmed)
	if abs, err := filepath.Abs(clean); err == nil {
		clean = abs
	}
	cmdExe := strings.TrimSpace(os.Getenv("ComSpec"))
	if cmdExe == "" {
		cmdExe = "cmd.exe"
	}
	cmd := exec.Command(cmdExe)
	cmd.Dir = filepath.Dir(clean)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_CONSOLE,
		CmdLine:       buildBatchCmdLine(cmdExe, clean),
	}
	return cmd.Start()
}

func resolveDirectoryTarget(dirPath string) (string, error) {
	trimmed := strings.TrimSpace(dirPath)
	if trimmed == "" {
		return "", fmt.Errorf("目录路径为空")
	}
	clean := filepath.Clean(trimmed)
	if abs, err := filepath.Abs(clean); err == nil {
		clean = abs
	}
	st, err := os.Stat(clean)
	if err != nil {
		return "", fmt.Errorf("打开目录失败 (%s): %w", clean, err)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("目标路径不是目录: %s", clean)
	}
	return clean, nil
}

func resolveRevealTarget(filePath string) (target string, selectItem bool, err error) {
	trimmed := strings.TrimSpace(filePath)
	if trimmed == "" {
		return "", false, fmt.Errorf("文件路径为空")
	}
	clean := filepath.Clean(trimmed)
	if abs, absErr := filepath.Abs(clean); absErr == nil {
		clean = abs
	}
	st, statErr := os.Stat(clean)
	if statErr == nil {
		if !st.IsDir() {
			return clean, true, nil
		}
		return clean, false, nil
	}

	parent := filepath.Dir(clean)
	if parent != "" && !SamePath(parent, clean) {
		if pst, pErr := os.Stat(parent); pErr == nil && pst.IsDir() {
			return parent, false, nil
		}
	}
	return "", false, fmt.Errorf("文件及所在目录不存在 (%s): %w", clean, statErr)
}

func ensureTrailingBackslash(dir string) string {
	if dir == "" || strings.HasSuffix(dir, `\`) {
		return dir
	}
	return dir + `\`
}

func explorerSelectParams(absPath string) string {
	return `/select,"` + absPath + `"`
}

func explorerExePath() string {
	winDir := os.Getenv("SystemRoot")
	if winDir == "" {
		winDir = os.Getenv("WINDIR")
	}
	if winDir != "" {
		candidate := filepath.Join(winDir, "explorer.exe")
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate
		}
	}
	return "explorer.exe"
}

func withCOMSTA(fn func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	hr32 := uint32(hr)
	switch hr32 {
	case sOK, sFalse:
		defer procCoUninitialize.Call()
	case rpcEChangedMode:
		return fmt.Errorf("COM 已以非 STA 模式初始化 (0x%08x)", hr32)
	default:
		return fmt.Errorf("CoInitializeEx 失败 (0x%08x)", hr32)
	}
	return fn()
}

func openDirectoryResolved(absDir string) error {
	var err error
	func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
		if uint32(hr) == sOK || uint32(hr) == sFalse {
			defer procCoUninitialize.Call()
		}
		err = shellExecuteW("open", ensureTrailingBackslash(absDir), "", "")
	}()
	if err == nil {
		return nil
	}
	cmd := exec.Command(explorerExePath(), absDir)
	return cmd.Start()
}

func shOpenFolderAndSelectItem(absPath string) error {
	return withCOMSTA(func() error {
		pathPtr, err := windows.UTF16PtrFromString(absPath)
		if err != nil {
			return err
		}
		pidl, _, _ := procILCreateFromPathW.Call(uintptr(unsafe.Pointer(pathPtr)))
		if pidl == 0 {
			return fmt.Errorf("ILCreateFromPathW 返回空 PIDL: %s", absPath)
		}
		defer procILFree.Call(pidl)

		hr, _, _ := procSHOpenFolderAndSelectItems.Call(pidl, 0, 0, 0)
		if int32(uint32(hr)) < 0 {
			return fmt.Errorf("SHOpenFolderAndSelectItems 失败 (0x%08x)", uint32(hr))
		}
		return nil
	})
}

// OpenDirectory 使用资源管理器打开目录。
func OpenDirectory(dirPath string) error {
	absDir, err := resolveDirectoryTarget(dirPath)
	if err != nil {
		return err
	}
	return openDirectoryResolved(absDir)
}

// RevealInExplorer 在资源管理器中选中指定文件；若为目录或文件不存在则打开对应目录。
func RevealInExplorer(filePath string) error {
	target, selectItem, err := resolveRevealTarget(filePath)
	if err != nil {
		return err
	}
	if !selectItem {
		return openDirectoryResolved(target)
	}
	if err := shOpenFolderAndSelectItem(target); err == nil {
		return nil
	}
	var shellErr error
	func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
		if uint32(hr) == sOK || uint32(hr) == sFalse {
			defer procCoUninitialize.Call()
		}
		shellErr = shellExecuteW("open", explorerExePath(), explorerSelectParams(target), "")
	}()
	if shellErr == nil {
		return nil
	}
	return openDirectoryResolved(filepath.Dir(target))
}

// StartMenuDir 获取并创建绿驿管家在 Windows 开始菜单中的程序目录。
func StartMenuDir() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		cfgDir, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("无法确定 APPDATA 目录: %w", err)
		}
		appData = cfgDir
	}
	dir := filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "绿驿管家")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// escapePSQuote 转义 PowerShell 单引号字符串中的单引号。
func escapePSQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// CreateShortcut 创建 Windows 快捷方式 (.lnk)，正确转义路径并设置工作目录。
func CreateShortcut(targetPath, shortcutPath, description string) error {
	if description == "" {
		base := filepath.Base(targetPath)
		description = strings.TrimSuffix(base, filepath.Ext(base))
	}
	workDir := filepath.Dir(targetPath)
	ps := fmt.Sprintf(
		"$ws = New-Object -ComObject WScript.Shell; $s = $ws.CreateShortcut('%s'); $s.TargetPath = '%s'; $s.WorkingDirectory = '%s'; $s.Description = '%s'; $s.Save();",
		escapePSQuote(shortcutPath),
		escapePSQuote(targetPath),
		escapePSQuote(workDir),
		escapePSQuote(description),
	)
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", ps)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("创建快捷方式失败: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Read7ZipRegistryPaths 从 64 位、32 位及当前用户注册表读取 7-Zip 安装目录。
func Read7ZipRegistryPaths() []RegistryResult {
	specs := []struct {
		label string
		root  registry.Key
		path  string
	}{
		{"HKLM\\SOFTWARE\\7-Zip (64位)", registry.LOCAL_MACHINE, `SOFTWARE\7-Zip`},
		{"HKLM\\SOFTWARE\\WOW6432Node\\7-Zip (32位)", registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\7-Zip`},
		{"HKCU\\SOFTWARE\\7-Zip (当前用户)", registry.CURRENT_USER, `SOFTWARE\7-Zip`},
	}

	results := make([]RegistryResult, 0, len(specs))
	for _, sp := range specs {
		k, err := registry.OpenKey(sp.root, sp.path, registry.QUERY_VALUE)
		if err != nil {
			results = append(results, RegistryResult{Label: sp.label, Err: err.Error()})
			continue
		}
		val, _, err := k.GetStringValue("Path")
		if err != nil {
			val, _, err = k.GetStringValue("Path64")
		}
		_ = k.Close()
		if err != nil {
			results = append(results, RegistryResult{Label: sp.label, Err: err.Error()})
			continue
		}
		dir := strings.TrimSpace(val)
		exe := filepath.Join(dir, "7z.exe")
		_, statErr := os.Stat(exe)
		results = append(results, RegistryResult{
			Label:   sp.label,
			DirPath: dir,
			ExePath: exe,
			Exists:  statErr == nil,
		})
	}
	return results
}

// RegistryResult 表示单条 7-Zip 注册表查询结果。
type RegistryResult struct {
	Label   string
	DirPath string
	ExePath string
	Exists  bool
	Err     string
}
