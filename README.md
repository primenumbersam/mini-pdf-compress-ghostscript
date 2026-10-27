# Mini PDF Compressor

외부 런타임(Python, Docker 등) 설치 없이 단일 실행 파일 더블클릭만으로 동작하는 로컬 웹 기반 Ghostscript PDF 압축 도구이다.

## 1. 아키텍처 특징 (Key Architecture)
- 자체 압축 해제형 단일 실행 바이너리 (Self-extracting Single Binary): Go의 `//go:embed` 기능을 활용하여 Ghostscript 엔진 및 단일 웹 UI(HTML/CSS/JS)를 단일 실행 파일에 패키징.
- 빌드 분기 최적화: `//go:build linux`, `//go:build windows` 빌드 태그를 사용하여 타깃 OS에 필요한 바이너리만 선별 임베딩(Linux 6MB, Windows 30MB).
- 무설치 라이프사이클: 실행 시 `os.TempDir()` 하위에 임시 폴더를 생성하여 엔진을 추출(Linux 0755 권한 부여)하고, `127.0.0.1:0` 동적 포트 바인딩 후 기본 브라우저를 자동 오픈.
- 자원 자동 정리: 프로그램 종료 시그널(SIGINT, SIGTERM) 수신 또는 UI 종료 버튼 클릭 시 임시 디렉터리를 일괄 삭제(`defer os.RemoveAll`).
- 극단 파일 안전 분기 (Fallback): 타임아웃(30초) 초과, 비정상 객체 오류, 또는 압축 후 용량 증가 시 원본 파일을 손상 없이 그대로 보존하여 제공.

## 2. 디렉터리 구조 (Flat Layout)

```
mini-pdf-compress-ghostscript/
├── main.go               # 서버 구동, HTTP 핸들러, 브라우저 기동 및 종료 라이프사이클
├── compress.go           # Ghostscript 파라미터 매핑, 프로세스 제어 및 Fallback 로직
├── engine.go             # 임시 디렉터리 생성 및 자원 정리 공통 로직
├── engine_linux.go       # Linux gs 바이너리 임베딩 및 실행 권한 부여
├── engine_windows.go     # Windows gswin64c.exe 및 gsdll64.dll 임베딩
├── browser.go            # OS별 기본 브라우저 자동 호출
├── index.html            # 단일 프론트엔드 UI (모노크롬 테마, //go:embed)
├── bin/
│   ├── linux/
│   │   └── gs            # Linux 포터블 실행 파일
│   └── windows/
│       ├── gswin64c.exe  # Windows Ghostscript 실행 파일
│       └── gsdll64.dll   # Windows 필수 라이브러리
├── dist/                 # 컴파일된 최종 단일 바이너리 저장소
├── go.mod
├── Makefile              # 크로스 플랫폼 빌드 스크립트
└── README.md             # 본 개발 가이드 문서
```

## 3. 빌드 가이드 (Build Guide)

### 3.1 사전 요구사항
- Go 1.22 이상 설치 환경.

### 3.2 일괄 빌드 (Makefile)
프로젝트 루트에서 Makefile 타깃을 실행한다.

```bash
# Linux 및 Windows 바이너리 일괄 빌드
make

# Linux 전용 빌드 (dist/mini-pdf-compress-linux-amd64)
make build-linux

# Windows 전용 빌드 (dist/mini-pdf-compress-windows-amd64.exe)
make build-windows

# 빌드 결과물 정리
make clean
```

### 3.3 수동 CLI 빌드

- Linux amd64 바이너리:
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/mini-pdf-compress-linux-amd64 .
chmod +x dist/mini-pdf-compress-linux-amd64
```

- Windows amd64 실행 파일 (콘솔 창 없는 GUI 모드):
```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -H=windowsgui" -o dist/mini-pdf-compress-windows-amd64.exe .
```

## 4. 실행 및 테스트 (Usage)

### 4.1 실행 방법

- 일반 사용자 실행 (더블 클릭):
  빌드된 바이너리(`dist/mini-pdf-compress-linux-amd64` 또는 `dist/mini-pdf-compress-windows-amd64.exe`)를 파일 관리자에서 더블 클릭한다. 백그라운드에서 로컬 서버가 시작되며 기본 브라우저에 압축 화면이 자동으로 열린다.

- 터미널 직접 실행:
```bash
./dist/mini-pdf-compress-linux-amd64
```

- 개발자 소스 실행 (Go 설치 환경):
```bash
go run .
```

### 4.2 압축 프리셋 사양
- Screen (`/screen`): 72 dpi 웹/스크린용 고압축 (기본값).
- eBook (`/ebook`): 150 dpi 일반 문서/가독성 균형 압축.
- Printer (`/printer`): 300 dpi 인쇄 품질 유지.
- Prepress (`/prepress`): 300 dpi 사전 인쇄 색상 보존.

### 4.3 프로그램 종료
- 브라우저 상단의 [종료] 버튼 클릭 시 `/api/shutdown`이 호출되며 서버와 임시 파일이 정리된다.
- 터미널에서 Ctrl+C를 입력해도 시그널 감지를 통해 임시 폴더가 삭제된다.
