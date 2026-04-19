command_not_found_handler() {
  # If wut isn't on PATH, fall through cleanly.
  # Without this guard, an un-PATHed binary would recurse this handler.
  if ! command -v wut >/dev/null 2>&1; then
    print -u2 "zsh: command not found: $1"
    return 127
  fi

  wut detect --line "$*"
  local rc=$?

  # Exit-code contract: 127 means "not natural language, let zsh print its
  # normal command-not-found message". Any other code means wut
  # (or the harness it exec'd into) handled the line — propagate that code.
  if [ $rc -eq 127 ]; then
    print -u2 "zsh: command not found: $1"
    return 127
  fi
  return $rc
}
