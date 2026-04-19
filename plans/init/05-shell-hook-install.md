# 05 — Shell hook install

## Description
Actually wire the snippet into a live zsh and confirm the round-trip: an unresolved first-token runs our handler, our handler invokes `wut detect`, and the exit code of detect controls whether the user sees `command not found`.

No code changes in this step — it's an integration checkpoint.

## Status
done

## Depends on
[03 — `init zsh` command](03-init-zsh-command.md), [04 — `detect` stub](04-detect-stub.md)

## What / where to change

Install the binary on `PATH` and source the snippet:

```
# install binary to a location on PATH
go install ./cmd/wut

# or, for iteration, symlink the repo build
ln -sf "$PWD/wut" /usr/local/bin/wut
```

Wire the hook into the current shell only (do **not** touch `~/.zshrc` yet — use a sandbox rc):

```
# write a sandbox rc
cat > /tmp/th-test.zshrc <<'EOF'
source ~/.zshrc 2>/dev/null || true
eval "$(wut init zsh)"
PROMPT='th-test$ '
EOF

# launch an isolated zsh using that rc
ZDOTDIR=/tmp zsh -i
# inside: rename the rc when entering ZDOTDIR/.zshrc, OR just:
zsh -c 'source /tmp/th-test.zshrc; exec zsh -i'
```

Simpler, if you're comfortable: `eval "$(wut init zsh)"` in your current interactive shell for the duration of testing. Undo by opening a new terminal.

## How to verify

In the hooked shell:

```
foobar
# stderr (from detect): wut: detect received: foobar
# stderr (from handler fallthrough): zsh: command not found: foobar
# exit: 127

ls
# runs ls normally — handler never fires for a resolvable command

which command_not_found_handler
# prints the function body — confirms the hook is installed
```

Pass criteria:
1. `foobar` shows **both** the detect log AND the command-not-found message (proves detect fired AND exit 127 fell through correctly).
2. `ls` behaves identically to a non-hooked shell (proves we don't intercept real commands).
3. No lag perceptible on real commands.

If `foobar` shows the detect log but no `command not found`, the snippet isn't returning 127 on fall-through — revisit step 03.
If `foobar` shows `command not found` but no detect log, the handler isn't calling `wut` — check PATH and snippet.
