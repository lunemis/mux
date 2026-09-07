# mux

**실시간 프리뷰를 제공하는 빠른 tmux 세션 전환기.**

tmux 세션, 윈도우, 페인을 터미널에서 빠르게 탐색하고 관리하는 TUI 도구입니다.

[English](README.md)

![Go](https://img.shields.io/badge/Go-1.24.2+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-blue.svg)

## 기능

- **전체 화면 실시간 프리뷰** — 선택한 세션·윈도우·페인의 터미널 출력을 전체 mux 화면에 표시 (500ms 주기 갱신). 가운데의 작은 선택 창에서 `l`/`→`/`Tab`으로 하위 단계에 들어가고 `h`/`←`/`Shift+Tab`으로 돌아가기
- **일반 명령어 표시** — tmux가 보고한 명령어를 별도의 프로그램별 연동 없이 세션과 페인 행에 표시
- **Git 브랜치 표시** — 각 세션의 현재 Git 브랜치를 표시하고 연결된 worktree를 구분
- **팝업 오버레이** — 실행 중인 프로그램과 관계없이 키 하나로 mux를 띄워 세션 전환
- **세션 관리** — TUI 내에서 생성/삭제/이름 변경
- **퀵 필터** — `/` 키로 세션 이름 또는 경로를 실시간 필터링

## 빠른 시작

```bash
# 인터랙티브 설치 (추천)
curl -sSL https://raw.githubusercontent.com/lunemis/mux/main/install.sh | bash

# 또는 직접 설치
brew install lunemis/tap/mux   # or: go install github.com/lunemis/mux/cmd/mux@latest
mux                             # 세션 매니저 실행
```

팝업 모드 설정 (tmux 위에 오버레이로 띄우기):

```bash
mux setup-keybind               # prefix + m을 바인딩하고 리로드 명령 출력
```

출력된 리로드 명령을 실행하세요. 이제 tmux에서 `Ctrl+b` → `m`으로 mux를 열 수 있습니다.

## 설치

### 인터랙티브 설치 (추천)

바이너리 설치와 키바인딩 설정을 안내합니다:

```bash
curl -sSL https://raw.githubusercontent.com/lunemis/mux/main/install.sh | bash
```

### Homebrew

```bash
brew install lunemis/tap/mux
```

### 소스에서 빌드

```bash
git clone https://github.com/lunemis/mux.git
cd mux
make install   # /usr/local/bin에 설치
```

### Go install

```bash
go install github.com/lunemis/mux/cmd/mux@latest
```

## 사용법

### 기본

`mux`를 실행하면 세션 매니저가 열립니다. `j`/`k`로 탐색하고 `Enter` 또는 `Backspace`로 attach하며 `q`로 종료합니다.

맨 위 한 줄에는 현재 단계에 따라 `tmux session picker`, `tmux window picker`, `tmux pane picker` 제목이 가운데 표시됩니다. 선택한 대상의 **실시간 프리뷰**는 별도의 바깥 테두리나 구분선 없이 나머지 모든 행을 채우며 500ms마다 갱신됩니다. 가운데에는 현재 계층 한 단계만 보여 주는 작은 선택 창이 떠 있습니다. 프롬프트와 상태 행이 잘 보이도록 프리뷰는 왼쪽 아래를 기준으로 유지되며, `?`를 누르면 상황별 키 도움말을 열고 닫을 수 있습니다.

세션은 OS 창 전환기처럼 동작합니다. tmux 안에서는 mux를 호출한 현재 세션이 첫 번째에, 직전에 사용한 세션이 두 번째에 표시되며 두 번째 세션이 처음부터 선택됩니다. 그 뒤에는 이전 세션들이 MRU 순서로 이어집니다. 바로 `Enter` 또는 `Backspace`를 누르면 선택된 직전 세션으로 전환되고, mux를 다시 열면 방금 떠난 세션이 선택됩니다. 백그라운드 출력은 목록 순서를 바꾸지 않습니다. tmux 밖에서는 MRU 우선 순서를 유지합니다. 한 번도 방문하지 않은 세션은 생성 시간(최신 우선), 그다음 이름 순으로 정렬됩니다.

이 전환기 동작 이전에 mux로 만든 팝업 키바인딩은 호출한 원래 세션을 팝업에 전달하도록 한 번 다시 생성해야 합니다. `setup-keybind`가 출력하는 리로드 명령을 실행하세요:

```bash
mux setup-keybind
```

직접 관리하는 팝업 키바인딩은 `mux`를 바로 실행하지 말고 호출한 세션을 전달하여 `mux popup`을 실행해야 합니다. 다음은 전역 `Ctrl+Backspace` 바인딩 예시입니다:

```tmux
bind-key -n C-BSpace run-shell 'MUX_ORIGIN_SESSION=#{q:session_name} "/absolute/path/to/mux" popup'
```

### 테마

mux에는 두 가지 내장 테마가 있습니다:

- `default` — 기존 어두운 터미널 팔레트
- `solarized-gruvbox` — Solarized 대비와 Gruvbox Light Soft 색상을 조합한 밝은 테마

`--theme`으로 테마를 선택할 수 있습니다:

```bash
mux --theme solarized-gruvbox
mux --theme solarized-gruvbox popup
```

`$XDG_CONFIG_HOME/mux/config.json`(`XDG_CONFIG_HOME`이 없으면 `~/.config/mux/config.json`)에 저장하려면:

```json
{
  "theme": "solarized-gruvbox"
}
```

현재 환경에만 적용하려면 `MUX_THEME=solarized-gruvbox`를 설정하세요. 우선순위는 `--theme`, `MUX_THEME`, XDG 설정, `default` 순입니다.

테마 팔레트는 [`theme/*.json`](theme/)에 있으며 바이너리에 내장됩니다. 새 내장 테마를 추가할 때는 기존 파일을 복사해 고유한 `name`과 모든 UI 의미 색상을 지정하세요. 터미널 배경을 유지하려면 `colors.background`를 `"NONE"`으로 설정합니다. 선택 창의 제목이 있는 위쪽 테두리는 `colors.separator`를 사용하며 나머지 `colors.border` 테두리와 독립적입니다.

### 커스텀 키바인딩

모든 mux 동작은 XDG `config.json`에서 키를 변경할 수 있습니다. 설정하지 않은 동작은 기본 키를 유지하고, 설정한 동작의 기본 키는 지정한 키 목록으로 교체됩니다. 각 동작에는 하나 이상의 키를 지정할 수 있습니다.

```json
{
  "theme": "solarized-gruvbox",
  "keybindings": {
    "global": {
      "quit": ["ctrl+q"]
    },
    "list": {
      "up": ["w", "up"],
      "down": ["s", "down"],
      "create": ["c"]
    },
    "create": {
      "submit": ["ctrl+s"],
      "cancel": ["ctrl+x"]
    }
  }
}
```

키 이름은 Bubble Tea 형식을 사용하며 대소문자를 구분합니다. 예: `enter`, `backspace`, `esc`, `tab`, `shift+tab`, `up`, `right`, `ctrl+c` 또는 문자 하나. 알 수 없는 컨텍스트/동작, 빈 키 목록, 같은 모드에서 충돌하는 키는 오류로 처리됩니다. 상황별 도움말 창과 확인 문구에는 현재 설정된 키가 표시됩니다.

| 컨텍스트 | 동작 | 기본 키 |
|---|---|---|
| `global` | `quit` | `ctrl+c` |
| `list` | `up` / `down` | `up`, `k` / `down`, `j` |
| `list` | `first` / `last` | `g` / `G` |
| `list` | `expand` / `collapse` | `tab`, `right`, `l` / `shift+tab`, `left`, `h` |
| `list` | `attach` | `enter`, `backspace` |
| `list` | `help` | `?` |
| `list` | `create` / `rename` / `kill` | `n` / `r` / `x` |
| `list` | `move_window` | `m` |
| `list` | `filter` / `clear_filter` | `/` / `esc` |
| `list` | `quit` | `q` |
| `create` | `switch_field` | `tab`, `shift+tab` |
| `create` | `submit` / `cancel` | `enter` / `esc` |
| `rename` | `submit` / `cancel` | `enter` / `esc` |
| `filter` | `apply` / `clear` | `enter` / `esc` |
| `kill` | `confirm` / `cancel` | `y`, `Y` / `any` |
| `move` | `up` / `down` | `up`, `k` / `down`, `j` |
| `move` | `confirm` / `cancel` | `enter` / `esc` |

`any`는 `kill.cancel`에서만 사용할 수 있는 기본 폴백입니다. `["n", "esc"]`처럼 명시적인 키로 교체하면 해당 키만 취소에 사용됩니다. 이 설정은 mux TUI 내부 키만 변경합니다. tmux 팝업 키는 `mux setup-keybind`로 별도 설정합니다.

윈도우를 이동하려면 세션 안으로 들어가 윈도우를 선택한 뒤 `m`을 누르세요. 대상 세션을 선택하고 `Enter`로 이동하거나 `Esc`로 취소합니다. 대상 세션의 활성 윈도우는 유지되고 비어 있는 다음 인덱스가 사용됩니다. 세션의 마지막 윈도우도 이동할 수 있으며, 선택 창에는 비게 된 소스 세션이 tmux에서 제거되고 연결된 클라이언트가 분리될 수 있다는 경고가 표시됩니다.

### 팝업 모드 (추천)

tmux 안에서 작업 중일 때, 어떤 프로그램이 실행 중이어도 키 하나로 mux를 테두리 없는 전체 화면 작업 전환기로 띄울 수 있습니다.

```bash
# 키바인딩 설정 (최초 1회)
mux setup-keybind          # prefix + m (기본값)
mux setup-keybind Space    # 다른 키로 변경 가능
```

`setup-keybind`가 출력하는 리로드 명령을 실행하세요. `mux popup`으로 수동 실행도 가능합니다.

> **참고:** tmux 3.2 이상 필요

### 기본 키바인딩

아래 기본 키는 [커스텀 키바인딩](#커스텀-키바인딩) 설정으로 교체할 수 있습니다.

| 키 | 동작 |
|---|---|
| `j` / `k` | 아래로 / 위로 이동 |
| `g` / `G` | 처음 / 마지막으로 이동 |
| `Tab` / `→` / `l` | 세션 → 윈도우 → 페인으로 들어가기 |
| `Shift+Tab` / `←` / `h` | 상위 단계로 돌아가기 |
| `Enter` / `Backspace` | attach (선택한 윈도우·페인까지 포커스) |
| `?` | 상황별 도움말 열기/닫기 |
| `n` | 새 세션 생성 |
| `r` | 세션 이름 변경 |
| `x` | 세션 삭제 (확인 후) |
| `m` | 선택한 윈도우를 다른 세션으로 이동 |
| `/` | 세션 필터링 |
| `Esc` | 필터 초기화 / 모드 취소 |
| `q` | 종료 |

## 요구사항

- tmux (팝업 모드는 3.2+)
- Linux 또는 macOS

## 기여

[CONTRIBUTING.md](CONTRIBUTING.md)를 참고하세요.

## 라이선스

[MIT](LICENSE)
