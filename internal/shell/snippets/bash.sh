# command_not_found_handle requires bash 4.0+. macOS ships bash 3.2 by
# default; users there must `brew install bash` and set that as their shell
# (or at minimum the interactive shell) for this hook to fire.
if [ "${BASH_VERSINFO[0]:-0}" -lt 4 ]; then
  return 0 2>/dev/null
fi

command_not_found_handle() {
  # If wut isn't on PATH, fall through cleanly.
  if ! command -v wut >/dev/null 2>&1; then
    printf "bash: %s: command not found\n" "$1" >&2
    return 127
  fi

  wut detect --line "$*"
  local rc=$?

  # Exit-code contract: 127 means "not natural language, let bash print its
  # normal command-not-found message". Any other code means wut
  # (or the harness it exec'd into) handled the line — propagate that code.
  if [ $rc -eq 127 ]; then
    printf "bash: %s: command not found\n" "$1" >&2
    return 127
  fi
  return $rc
}
