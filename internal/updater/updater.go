package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"carrydock/internal/archive"
	"carrydock/internal/logger"
	"carrydock/internal/version"
	"github.com/egoist/mygo/netproxy"
)

const defaultGitHubReleaseURL = "https://api.github.com/repos/youzhiran/CarryDock/releases/latest"

// ReleaseInfo 描述远程最新发行版信息。
type ReleaseInfo struct {
	Version     string
	Changelog   string
	DownloadURL string
	AssetName   string
}

// Service 负责 GitHub Release 版本检查、更新包下载（支持取消与系统代理）及安全替换安装。
type Service struct {
	client  *http.Client
	repoURL string
}

// NewService 创建一个自动跟随系统代理的更新服务实例。
func NewService() *Service {
	return &Service{
		client: &http.Client{
			Transport: netproxy.Transport(),
			Timeout:   30 * time.Second,
		},
		repoURL: defaultGitHubReleaseURL,
	}
}

// CheckForUpdate 仅发起一次网络请求获取远程版本并与当前版本比对。
// 修复了原版一次检查请求两次 GitHub API、以及无 +buildNumber 后缀时无法判断更新的问题。
func (s *Service) CheckForUpdate(ctx context.Context, forceReinstall bool) (hasUpdate bool, info *ReleaseInfo, err error) {
	info, err = s.fetchLatestRelease(ctx)
	if err != nil {
		return false, nil, err
	}
	if forceReinstall {
		return true, info, nil
	}
	return IsNewerVersion(version.FullVersion(), info.Version), info, nil
}

func (s *Service) fetchLatestRelease(ctx context.Context) (*ReleaseInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.repoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "CarryDock/"+version.Version)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("连接 GitHub 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		var errPayload struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &errPayload) == nil && errPayload.Message != "" {
			return nil, fmt.Errorf("GitHub HTTP %d: %s", resp.StatusCode, errPayload.Message)
		}
		return nil, fmt.Errorf("GitHub HTTP %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Body    string `json:"body"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("解析 GitHub Release 响应失败: %w", err)
	}

	for _, asset := range release.Assets {
		lower := strings.ToLower(asset.Name)
		if strings.HasSuffix(lower, ".zip") || strings.Contains(lower, "windows") {
			return &ReleaseInfo{
				Version:     release.TagName,
				Changelog:   release.Body,
				DownloadURL: asset.BrowserDownloadURL,
				AssetName:   asset.Name,
			}, nil
		}
	}
	return nil, fmt.Errorf("GitHub Release (%s) 中未找到 Windows 更新包", release.TagName)
}

var buildNumberRe = regexp.MustCompile(`\+(\d+)$`)

// IsNewerVersion 比较本地与远程版本号；同时支持 +buildNumber 及 SemVer (x.y.z)。
func IsNewerVersion(localVer, remoteVer string) bool {
	lm := buildNumberRe.FindStringSubmatch(strings.TrimSpace(localVer))
	rm := buildNumberRe.FindStringSubmatch(strings.TrimSpace(remoteVer))
	if len(lm) == 2 && len(rm) == 2 {
		lb, _ := strconv.Atoi(lm[1])
		rb, _ := strconv.Atoi(rm[1])
		if rb != lb {
			return rb > lb
		}
	}
	return compareSemver(remoteVer, localVer) > 0
}

func parseSemver(v string) [3]int {
	v = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(v), "v"))
	if idx := strings.IndexAny(v, "+-"); idx >= 0 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	var out [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		out[i], _ = strconv.Atoi(parts[i])
	}
	return out
}

func compareSemver(a, b string) int {
	pa, pb := parseSemver(a), parseSemver(b)
	for i := range 3 {
		if pa[i] != pb[i] {
			if pa[i] > pb[i] {
				return 1
			}
			return -1
		}
	}
	return 0
}

// DownloadUpdate 下载更新包到 savePath，通过 ctx 支持随时取消，并实时回调下载进度 [0, 1]。
func (s *Service) DownloadUpdate(ctx context.Context, downloadURL, savePath string, onProgress func(progress float64)) error {
	dlClient := &http.Client{
		Transport: netproxy.Transport(),
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "CarryDock/"+version.Version)

	resp, err := dlClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}

	out, err := os.OpenFile(savePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}

	buf := make([]byte, 32*1024)
	var received int64
	total := resp.ContentLength
	for {
		if err := ctx.Err(); err != nil {
			_ = out.Close()
			_ = os.Remove(savePath)
			return err
		}
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				_ = out.Close()
				_ = os.Remove(savePath)
				return writeErr
			}
			received += int64(n)
			if total > 0 && onProgress != nil {
				p := float64(received) / float64(total)
				if p > 1 {
					p = 1
				}
				onProgress(p)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = out.Close()
			_ = os.Remove(savePath)
			return readErr
		}
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(savePath)
		return err
	}
	if onProgress != nil {
		onProgress(1)
	}
	return nil
}

// InstallUpdate 安全解压更新包、生成带安全引号且保护 data/ 目录的更新脚本，并拉起更新后退出当前进程。
// 修复了原版用 configFilePath.indexOf('data') 截断目录的致命 bug、Zip-Slip 漏洞以及配置文件不存在时崩溃的问题。
func (s *Service) InstallUpdate(updateZipPath string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("无法确定当前可执行文件路径: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolved
	}
	appDir := filepath.Dir(exePath)
	exeName := filepath.Base(exePath)

	tempDir, err := os.MkdirTemp("", "carrydock_update_")
	if err != nil {
		return err
	}
	extractDir := filepath.Join(tempDir, "pkg")
	if err := archive.Extract(updateZipPath, extractDir, archive.FormatZip); err != nil {
		_ = os.RemoveAll(tempDir)
		return fmt.Errorf("解压更新包失败: %w", err)
	}

	sourceDir := findDirWithExe(extractDir)
	if sourceDir == "" {
		_ = os.RemoveAll(tempDir)
		return fmt.Errorf("更新包中未找到包含 .exe 可执行文件的目录")
	}
	// 如果新包中的主 exe 名称变更，优先采用新包的 exe 名称
	if entries, err := os.ReadDir(sourceDir); err == nil {
		for _, ent := range entries {
			if !ent.IsDir() && strings.EqualFold(filepath.Ext(ent.Name()), ".exe") {
				exeName = ent.Name()
				break
			}
		}
	}

	scriptPath := filepath.Join(tempDir, "update.bat")
	script := fmt.Sprintf(`@echo off
setlocal
title CarryDock Update
echo ============================================
echo CarryDock Auto Update
echo ============================================
set "appDir=%s"
set "srcDir=%s"
set "exeName=%s"

echo Waiting for CarryDock to exit...
timeout /t 2 /nobreak >nul

echo Copying new files (preserving data folder)...
robocopy "%%srcDir%%" "%%appDir%%" /E /XD data ~archives /R:3 /W:1 >nul
if %%errorlevel%% GEQ 8 (
    echo ERROR: Failed to copy update files.
    pause
    exit /b 1
)

echo Restarting CarryDock...
start "" "%%appDir%%\%%exeName%%"
cd /d "%%TEMP%%"
rmdir /s /q "%s" 2>nul
exit /b 0
`, appDir, sourceDir, exeName, tempDir)

	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		_ = os.RemoveAll(tempDir)
		return err
	}

	logger.Info("启动更新脚本: %s (appDir=%s, srcDir=%s)", scriptPath, appDir, sourceDir)
	cmd := exec.Command("cmd.exe", "/c", "start", "", scriptPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动更新脚本失败: %w", err)
	}
	os.Exit(0)
	return nil
}

func findDirWithExe(root string) string {
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found != "" {
			return nil
		}
		if !d.IsDir() && strings.EqualFold(filepath.Ext(d.Name()), ".exe") {
			found = filepath.Dir(path)
			return filepath.SkipAll
		}
		return nil
	})
	return found
}
