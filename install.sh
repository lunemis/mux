#!/usr/bin/env bash
set -euo pipefail

REPO="lunemis/mux"
BINARY="mux"
INSTALL_DIR="/usr/local/bin"
# Parse flags
for arg in "$@"; do
    case "$arg" in
        --help|-h)
            echo "Usage: install.sh"
            exit 0
            ;;
    esac
done

# --- Helpers ---

info()  { printf '\033[1;34m→\033[0m %s\n' "$*"; }
ok()    { printf '\033[1;32m✓\033[0m %s\n' "$*"; }
skip()  { printf '\033[1;33m⊘\033[0m %s\n' "$*"; }
warn()  { printf '\033[1;33m⚠\033[0m %s\n' "$*"; }
fail()  { printf '\033[1;31m✗\033[0m %s\n' "$*"; exit 1; }

ask() {
    local prompt="$1"
    local default="${2:-Y}"
    local yn
    if [ "$default" = "Y" ]; then
        printf '\033[1m%s\033[0m [Y/n] ' "$prompt"
    else
        printf '\033[1m%s\033[0m [y/N] ' "$prompt"
    fi
    read -r yn </dev/tty || yn=""
    yn="${yn:-$default}"
    case "$yn" in
        [Yy]*) return 0 ;;
        *) return 1 ;;
    esac
}

detect_platform() {
    local os arch
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    arch="$(uname -m)"

    case "$os" in
        linux)  OS="linux" ;;
        darwin) OS="darwin" ;;
        *)      fail "Unsupported OS: $os" ;;
    esac

    case "$arch" in
        x86_64|amd64)   ARCH="amd64" ;;
        arm64|aarch64)  ARCH="arm64" ;;
        *)              fail "Unsupported architecture: $arch" ;;
    esac
}

latest_version() {
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' \
        | head -1 \
        | sed -E 's/.*"([^"]+)".*/\1/'
}

# --- Steps ---

install_binary() {
    info "Installing ${BINARY}..."

    # Try go install first
    if command -v go &>/dev/null; then
        if ask "  Go detected. Use 'go install'?"; then
            go install "github.com/${REPO}/cmd/${BINARY}@latest"
            if command -v "${BINARY}" &>/dev/null; then
                ok "${BINARY} installed via go install"
            else
                # go install succeeded but the binary isn't on PATH.
                # This is common when GOPATH/bin (or GOBIN) is not in PATH.
                local gobin
                gobin="$(go env GOBIN)"
                [ -z "$gobin" ] && gobin="$(go env GOPATH)/bin"
                ok "${BINARY} installed to ${gobin}/${BINARY}"
                warn "${gobin} is not on your PATH"
                warn "  Add this to your shell config (e.g. ~/.zshrc or ~/.bashrc):"
                warn "  export PATH=\"${gobin}:\$PATH\""
            fi
            return
        fi
    fi

    # Fall back to GitHub Release download
    detect_platform
    local version
    version="$(latest_version)" || fail "Could not determine latest version"
    local name="${BINARY}_${version#v}_${OS}_${ARCH}"
    local url="https://github.com/${REPO}/releases/download/${version}/${name}.tar.gz"

    info "Downloading ${BINARY} ${version} (${OS}/${ARCH})..."
    # NOTE: tmp must NOT be `local` — the EXIT trap fires at script exit,
    # long after this function returned, and under `set -u` referencing a
    # vanished local aborts the script with "unbound variable" (exit 1).
    tmp="$(mktemp -d)"
    trap '[ -n "${tmp:-}" ] && rm -rf "$tmp"' EXIT

    curl -fsSL "$url" -o "${tmp}/${name}.tar.gz" \
        || fail "Download failed. Check https://github.com/${REPO}/releases"
    tar -xzf "${tmp}/${name}.tar.gz" -C "$tmp"

    # Determine install path
    if [ -w "$INSTALL_DIR" ]; then
        install -m 755 "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
        ok "${BINARY} installed to ${INSTALL_DIR}/${BINARY}"
    else
        local local_bin="${HOME}/.local/bin"
        mkdir -p "$local_bin"
        install -m 755 "${tmp}/${BINARY}" "${local_bin}/${BINARY}"
        ok "${BINARY} installed to ${local_bin}/${BINARY}"
        if ! echo "$PATH" | grep -q "$local_bin"; then
            warn "Add ${local_bin} to your PATH"
        fi
    fi
}

is_oh_my_tmux() {
    # Hybrid detection mirroring tmux/popup.go's isOhMyTmux:
    #   A) symlink whose target lives under a `.tmux/` directory.
    #      Match both absolute (`/home/u/.tmux/.tmux.conf`) and relative
    #      (`.tmux/.tmux.conf`) targets — the upstream installer uses the
    #      latter via `ln -s .tmux/.tmux.conf ~/.tmux.conf`.
    #   B) first line is the oh-my-tmux heredoc signature `# : << 'EOF'`
    local conf="$1"
    [ -e "$conf" ] || return 1
    if [ -L "$conf" ]; then
        local target
        target="$(readlink "$conf")"
        case "$target" in
            */.tmux/*|.tmux/*) return 0 ;;
        esac
    fi
    local first
    first="$(head -n1 "$conf" 2>/dev/null || true)"
    [ "$first" = "# : << 'EOF'" ]
}

# strip_mux_lines removes mux-owned bind lines from a config file:
#   - lines tagged with the marker `# mux popup keybinding`
#   - legacy untagged lines from older install.sh fallbacks: any line
#     containing `display-popup -E -w 80% -h 80% "mux"`
# Used to clean up the main .tmux.conf when routing to .tmux.conf.local.
# Writes through symlinks (oh-my-tmux's ~/.tmux.conf is normally a symlink to
# ~/.tmux/.tmux.conf — `mv` would replace the symlink, leaving the real file
# corrupt and breaking `~/.tmux/install.sh` reinstalls).
strip_mux_lines() {
    local target="$1"
    [ -f "$target" ] || return 0
    local marker='# mux popup keybinding'
    local legacy='display-popup -E -w 80% -h 80% "mux"'
    local tmp
    tmp="$(mktemp)"
    grep -vF -e "$marker" -e "$legacy" "$target" > "$tmp" || true
    if cmp -s "$target" "$tmp"; then
        rm -f "$tmp"
        return 1
    fi
    cat "$tmp" > "$target"
    rm -f "$tmp"
    return 0
}

setup_keybind() {
    info "Setting up tmux keybinding..."
    if command -v "$BINARY" &>/dev/null; then
        "$BINARY" setup-keybind m
        ok "Keybinding added: prefix + m → mux popup"
    else
        local conf=""
        local xdg="${XDG_CONFIG_HOME:-}"
        if [ -n "$xdg" ] && [ -f "${xdg}/tmux/tmux.conf" ]; then
            conf="${xdg}/tmux/tmux.conf"
        elif [ -f "${HOME}/.config/tmux/tmux.conf" ]; then
            conf="${HOME}/.config/tmux/tmux.conf"
        else
            conf="${HOME}/.tmux.conf"
        fi
        local line='bind-key m display-popup -E -w 80% -h 80% "mux"'
        local marker='# mux popup keybinding'

        # Route to .tmux.conf.local for oh-my-tmux users — the main conf gets
        # processed via `cut -c3- | sh`, and any non-`# `-prefixed line we add
        # there becomes invalid shell at reload time.
        if is_oh_my_tmux "$conf"; then
            local local_conf
            case "$conf" in
                */.tmux.conf) local_conf="${conf}.local" ;;
                */tmux.conf)  local_conf="$(dirname "$conf")/tmux.conf.local" ;;
                *)            local_conf="${conf}.local" ;;
            esac
            if [ -f "$local_conf" ] && grep -qF "$marker" "$local_conf"; then
                ok "Keybinding already exists in ${local_conf}"
            else
                local tagged="${line}  ${marker}"
                if [ -f "$local_conf" ] && grep -qF '# "$@"' "$local_conf"; then
                    # Insert before the sentinel line oh-my-tmux marks as off-limits.
                    awk -v ins="$tagged" '
                        /^# "\$@"[[:space:]]*$/ && !done { print ""; print ins; print ""; done=1 }
                        { print }
                    ' "$local_conf" > "${local_conf}.tmp" && mv "${local_conf}.tmp" "$local_conf"
                else
                    printf '\n%s\n' "$tagged" >> "$local_conf"
                fi
                ok "Detected oh-my-tmux. Keybinding added to ${local_conf}"
                # Best-effort cleanup of any prior corrupt entry in the main
                # conf — older mux versions appended unmarked binds there.
                if strip_mux_lines "$conf"; then
                    ok "Removed prior mux entry from ${conf}"
                fi
                if [ -n "${TMUX:-}" ]; then
                    tmux source-file "$local_conf" 2>/dev/null && ok "tmux config reloaded"
                fi
            fi
            return 0
        fi

        if [ -f "$conf" ] && grep -qF "$marker" "$conf"; then
            ok "Keybinding already exists in ${conf}"
        else
            printf '%s  %s\n' "$line" "$marker" >> "$conf"
            ok "Keybinding added to ${conf}"
            if [ -n "${TMUX:-}" ]; then
                tmux source-file "$conf" 2>/dev/null && ok "tmux config reloaded"
            fi
        fi
    fi
}

# --- Main ---

echo ""
echo "  ⚡ mux installer"
echo ""

# Step 1: Install binary
if ask "[1/2] Install ${BINARY} binary?"; then
    install_binary
else
    skip "${BINARY} binary installation skipped"
fi
echo ""

# Step 2: tmux keybinding
if ask "[2/2] Configure tmux keybinding (prefix+m)?"; then
    setup_keybind
else
    skip "Keybinding setup skipped"
fi

echo ""
echo "  Done! Run 'mux' in tmux to start."
echo ""
