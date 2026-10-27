package main

import (
	"log"
	"os"
)

// EngineManager는 압축 엔진의 임시 디렉터리 및 실행 바이너리 경로를 관리한다.
type EngineManager struct {
	TempDir    string
	BinaryPath string
}

// Cleanup은 임시 디렉터리와 내부 추출 파일들을 모두 제거한다.
func (e *EngineManager) Cleanup() {
	if e.TempDir != "" {
		if err := os.RemoveAll(e.TempDir); err != nil {
			log.Printf("임시 디렉터리 정리 실패 (%s): %v", e.TempDir, err)
		} else {
			log.Printf("임시 디렉터리 정리 완료: %s", e.TempDir)
		}
	}
}
