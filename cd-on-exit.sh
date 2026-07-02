fm() {
    fmtui 
    local dir
    dir=$(cat /tmp/fmtui-dir 2>/dev/null)
    [ -d "$dir" ] && cd "$dir" 
}