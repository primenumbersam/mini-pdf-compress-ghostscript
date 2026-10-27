package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// CompressResult는 PDF 압축 결과를 담는 구조체이다.
type CompressResult struct {
	ID             string  `json:"id"`
	Filename       string  `json:"filename"`
	OriginalSize   int64   `json:"originalSize"`
	CompressedSize int64   `json:"compressedSize"`
	Ratio          float64 `json:"ratio"`
	IsFallback     bool    `json:"isFallback"`
	OutputPath     string  `json:"-"`
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// CompressPDF는 Ghostscript 바이너리를 실행하여 PDF 파일을 압축한다.
// 타임아웃(30초) 초과, 엔진 오류, 또는 압축 후 용량 증가 시 원본 보존(Fallback)으로 분기한다.
func CompressPDF(binaryPath, inputPath, outputPath, preset string) (*CompressResult, error) {
	inInfo, err := os.Stat(inputPath)
	if err != nil {
		return nil, fmt.Errorf("입력 파일 확인 실패: %w", err)
	}
	originalSize := inInfo.Size()

	qualityMap := map[string]string{
		"screen":   "/screen",
		"ebook":    "/ebook",
		"printer":  "/printer",
		"prepress": "/prepress",
	}

	pdfSetting, exists := qualityMap[preset]
	if !exists {
		pdfSetting = "/screen"
	}

	// 30초 실행 제한 컨텍스트
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := []string{
		"-sDEVICE=pdfwrite",
		"-dCompatibilityLevel=1.4",
		"-dPDFSETTINGS=" + pdfSetting,
		"-dDownsampleMonoImages=false", // 모노크롬 다운샘플링 필터 오류 억제
		"-dNOPAUSE",
		"-dQUIET",
		"-dBATCH",
		"-sOutputFile=" + outputPath,
		inputPath,
	}

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	// 타임아웃 또는 엔진 오류 발생 시 원본 보존 fallback 실행
	if runErr != nil {
		_ = os.Remove(outputPath)
		if cpErr := copyFile(inputPath, outputPath); cpErr != nil {
			return nil, fmt.Errorf("원본 보존 복사 실패: %w (압축 오류: %v, stderr: %s)", cpErr, runErr, stderr.String())
		}
		return &CompressResult{
			OriginalSize:   originalSize,
			CompressedSize: originalSize,
			Ratio:          0.0,
			IsFallback:     true,
			OutputPath:     outputPath,
		}, nil
	}

	outInfo, err := os.Stat(outputPath)
	if err != nil {
		return nil, fmt.Errorf("출력 파일 확인 실패: %w", err)
	}

	compressedSize := outInfo.Size()

	// 압축 결과가 원본보다 크거나 같으면 원본 파일로 대체
	if compressedSize >= originalSize {
		_ = os.Remove(outputPath)
		if cpErr := copyFile(inputPath, outputPath); cpErr != nil {
			return nil, fmt.Errorf("원본 대체 복사 실패: %w", cpErr)
		}
		return &CompressResult{
			OriginalSize:   originalSize,
			CompressedSize: originalSize,
			Ratio:          0.0,
			IsFallback:     true,
			OutputPath:     outputPath,
		}, nil
	}

	var ratio float64
	if originalSize > 0 {
		ratio = (1.0 - float64(compressedSize)/float64(originalSize)) * 100.0
	}

	return &CompressResult{
		OriginalSize:   originalSize,
		CompressedSize: compressedSize,
		Ratio:          ratio,
		IsFallback:     false,
		OutputPath:     outputPath,
	}, nil
}
