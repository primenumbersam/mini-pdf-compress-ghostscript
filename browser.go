package main

import (
	"log"
	"os/exec"
	"runtime"
)

// OpenBrowser는 시스템 기본 브라우저를 백그라운드로 실행하여 지정된 URL을 연다.
func OpenBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux, freebsd 등
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Start(); err != nil {
		log.Printf("브라우저 자동 실행 실패: %v (수동으로 %s 에 접속)", err, url)
	}
}
