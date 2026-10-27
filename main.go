package main

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

//go:embed index.html
var indexHTML []byte

var (
	storageMu sync.RWMutex
	storage   = make(map[string]CompressResult)
)

func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func main() {
	engine, err := SetupEngine()
	if err != nil {
		log.Fatalf("엔진 초기화 실패: %v", err)
	}
	defer engine.Cleanup()

	log.Printf("엔진 준비 완료 (바이너리: %s, 임시 디렉터리: %s)", engine.BinaryPath, engine.TempDir)

	mux := http.NewServeMux()

	// 정적 웹 UI
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})

	// 압축 API
	mux.HandleFunc("POST /api/compress", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		// 최대 500MB 파싱 허용
		if err := r.ParseMultipartForm(500 << 20); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "요청 파싱 실패: " + err.Error()})
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "파일을 읽을 수 없다: " + err.Error()})
			return
		}
		defer file.Close()

		if !strings.HasSuffix(strings.ToLower(header.Filename), ".pdf") {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "PDF 파일만 업로드할 수 있다."})
			return
		}

		preset := r.FormValue("preset")
		if preset == "" {
			preset = "screen"
		}

		fileID := generateID()
		inputPath := filepath.Join(engine.TempDir, fmt.Sprintf("%s_in.pdf", fileID))
		outputPath := filepath.Join(engine.TempDir, fmt.Sprintf("%s_out.pdf", fileID))

		// 원본 파일 임시 디렉터리에 저장
		dstFile, err := os.Create(inputPath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "입력 임시 파일 생성 실패: " + err.Error()})
			return
		}

		if _, err := io.Copy(dstFile, file); err != nil {
			dstFile.Close()
			_ = os.Remove(inputPath)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "파일 저장 실패: " + err.Error()})
			return
		}
		dstFile.Close()
		defer os.Remove(inputPath) // 압축 완료 후 원본 임시 파일은 즉시 제거

		// Ghostscript 압축 실행
		result, err := CompressPDF(engine.BinaryPath, inputPath, outputPath, preset)
		if err != nil {
			_ = os.Remove(outputPath)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error": err.Error()})
			return
		}

		result.ID = fileID
		result.Filename = header.Filename

		storageMu.Lock()
		storage[fileID] = *result
		storageMu.Unlock()

		_ = json.NewEncoder(w).Encode(map[string]any{
			"success":        true,
			"id":             result.ID,
			"filename":       result.Filename,
			"originalSize":   result.OriginalSize,
			"compressedSize": result.CompressedSize,
			"ratio":          result.Ratio,
			"isFallback":     result.IsFallback,
		})
	})

	// 다운로드 API
	mux.HandleFunc("GET /api/download/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.NotFound(w, r)
			return
		}

		storageMu.RLock()
		item, exists := storage[id]
		storageMu.RUnlock()

		if !exists {
			http.NotFound(w, r)
			return
		}

		if _, err := os.Stat(item.OutputPath); os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}

		downloadFilename := strings.TrimSuffix(item.Filename, filepath.Ext(item.Filename)) + "_compressed.pdf"
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", downloadFilename))
		w.Header().Set("Content-Type", "application/pdf")
		http.ServeFile(w, r, item.OutputPath)
	})

	// 종료 API
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	mux.HandleFunc("POST /api/shutdown", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"shutting down"}`))
		go func() {
			time.Sleep(200 * time.Millisecond)
			stopChan <- syscall.SIGTERM
		}()
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("포트 바인딩 실패: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	serverURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	fmt.Printf("HTTP 서버 시작: %s\n", serverURL)

	// 브라우저 자동 오픈
	go func() {
		time.Sleep(150 * time.Millisecond)
		OpenBrowser(serverURL)
	}()

	server := &http.Server{Handler: mux}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("서버 실행 오류: %v", err)
		}
	}()

	sig := <-stopChan
	log.Printf("종료 시그널 수신 (%v). 서버 종료 중...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("서버 강제 종료: %v", err)
	}
}
