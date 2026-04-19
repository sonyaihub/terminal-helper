# Plan: `m5` — distribution

Goal: finish the install story so the project can actually reach users.
Ship shell integrations for bash and fish, a curl-friendly install script,
and a homebrew formula template.

| # | Step | Status |
|---|---|---|
| 01 | `init bash` — bash hook + embedded snippet | done |
| 02 | `init fish` — fish hook + embedded snippet | done |
| 03 | `scripts/install.sh` — curl | sh installer | done |
| 04 | `Formula/wut.rb` — homebrew template | done |
| 05 | Verification | done |

## Shell-hook notes

- **bash**: the function is `command_not_found_handle` (no trailing `r`).
  Args come as positional params; we join with `$*` just like zsh.
- **fish**: `function fish_command_not_found`; args are in `$argv`. Status
  is read with `$status` not `$?`. We emit the same 127-means-passthrough
  contract the zsh snippet uses.
