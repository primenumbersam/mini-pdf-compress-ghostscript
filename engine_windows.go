//go:build windows

package main

import (
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed bin/windows/*
var windowsBinFS embed.FS

// SetupEngine은 Windows 환경에서 Ghostscript 실행 바이너리를 준비한다.
func SetupEngine() (*EngineManager, error) {
	tempDir, err := os.MkdirTemp("", "pdfcompress-*")
	if err != nil {
		return nil, fmt.Errorf("임시 디렉터리 생성 실패: %w", err)
	}

	targetBinary := filepath.Join(tempDir, "gswin64c.exe")

	// 내장 바이너리 및 DLL 일괄 추출
	hasEmbedded := false
	entries, err := windowsBinFS.ReadDir("bin/windows")
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || entry.Name() == ".gitkeep" {
				continue
			}

			srcFile, err := windowsBinFS.Open("bin/windows/" + entry.Name())
			if err != nil {
				continue
			}

			destPath := filepath.Join(tempDir, entry.Name())
			destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				srcFile.Close()
				continue
			}

			_, _ = io.Copy(destFile, srcFile)
			srcFile.Close()
			destFile.Close()

			if entry.Name() == "gswin64c.exe" {
				hasEmbedded = true
			}
		}
	}

	if hasEmbedded {
		return &EngineManager{
			TempDir:    tempDir,
			BinaryPath: targetBinary,
		}, nil
	}

	// 시스템 PATH에 등록된 gswin64c 또는 gs 검색
	for _, name := range []string{"gswin64c.exe", "gswin64c", "gs.exe", "gs"} {
		if sysPath, err := exec.LookPath(name); err == nil {
			return &EngineManager{
				TempDir:    tempDir,
				BinaryPath: sysPath,
			}, nil
		}
	}

	_ = os.RemoveAll(tempDir)
	return nil, fmt.Errorf("Ghostscript 바이너리를 초기화할 수 없다 (내장 및 시스템 gs 없음)")
}
