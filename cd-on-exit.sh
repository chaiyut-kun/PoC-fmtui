fm() {
    local tmp="${TMPDIR:-/tmp}/fmtui-dir"
    command fm "$@"
    if [ -f "$tmp" ]; then
        local dir
        dir=$(cat "$tmp")
        rm -f "$tmp"
        [ -d "$dir" ] && cd "$dir"
    fi
}