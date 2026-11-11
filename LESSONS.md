# Lessons

What recent work got wrong, and the rule that would have caught it.

## A rename script must not rewrite itself

The brand rename script contained the old name and matched its own file, so
it changed its rules mid-run and later steps ran against mutated text.
Rule: keep a rename script out of its own glob, or run it from outside the
tree, and compare the result against the untouched source afterwards.

## Machine contracts are not brand strings

The rename also rewrote the `x-magpie-*` request headers, which community
plugins read, so session, work-dir and fallback no longer reached them.
Rule: before a rename, list the identifiers other programs read (npm scope,
plugin repo, request headers) and exclude them. Plugin-facing headers go out
under both `X-Jianmu-*` and `x-magpie-*` (`provider.SetPlugin`).

## A skipped test is not a passing test

`TestPluginToldTheSession` needs Bun 1.4.2 or newer and skips on older ones,
so a green run proved nothing. Rule: run with `-v`, read SKIP lines, and run
the test under a new enough Bun before calling the change verified.

## An id change orphans what the old id registered

Changing the bundle id and service label left the old launch agents behind,
the login item and the gateway service both. Rule: when an identifier moves,
add a migration for each record the system keeps under the old one, with a
test that fails without it.

## Do not ship credentials in source

GitHub push protection rejected the push for OAuth client secrets in
`internal/provider/google.go`. Rule: keep secrets out of source and history;
never bypass push protection.
