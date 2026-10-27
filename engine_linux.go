//go:build linux

package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed bin/linux/*
var linuxBinFS embed.FS

// SetupEngine은 Linux 환경에서 Ghostscript 실행 바이너리를 준비한다.
func SetupEngine() (*EngineManager, error) {
	tempDir, err := os.MkdirTemp("", "pdfcompress-*")
	if err != nil {
		return nil, fmt.Errorf("임시 디렉터리 생성 실패: %w", err)
	}

	targetBinary := filepath.Join(tempDir, "gs")
	embeddedFile, err := linuxBinFS.Open("bin/linux/gs")
	if err == nil {
		defer embeddedFile.Close()

		outFile, err := os.OpenFile(targetBinary, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err == nil {
			_, err = io.Copy(outFile, embeddedFile)
			outFile.Close()
			if err == nil {
				// 실행 테스트
				cmd := exec.Command(targetBinary, "--version")
				if err := cmd.Run(); err == nil {
					return &EngineManager{
						TempDir:    tempDir,
						BinaryPath: targetBinary,
					}, nil
				}
			}
		}
	}

	// 내장 바이너리 실행 불가 시 시스템 gs 검색
	if sysGS, err := exec.LookPath("gs"); err == nil {
		return &EngineManager{
			TempDir:    tempDir,
			BinaryPath: sysGS,
		}, nil
	}

	_ = os.RemoveAll(tempDir)
	return nil, fmt.Errorf("Ghostscript 바이너리를 초기화할 수 없다 (내장 및 시스템 gs 없음)")
}

func init() {
	// embed.FS 컴파일 검증
	_ = fs.WalkDir
}
