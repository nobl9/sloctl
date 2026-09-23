#!/usr/bin/env bash

set -e

TMP_DIR=$(mktemp -d)

cleanup_git() {
  git -C "$TMP_DIR" clean -df
  git -C "$TMP_DIR" checkout -- .
}

main() {
  cp -r . "$TMP_DIR"
  cleanup_git

  make -C "$TMP_DIR" generate

  TRACKED_CHANGED=$(git -C "$TMP_DIR" diff --name-only)
  UNTRACKED_CHANGED=$(git -C "$TMP_DIR" ls-files --others --exclude-standard)
  if [[ -z "${TRACKED_CHANGED}" && -z "${UNTRACKED_CHANGED}" ]]; then
    echo "Looks good!"
    return
  fi

  printf >&2 "There are generated changes that are not committed:\n"
  if [[ -n "${TRACKED_CHANGED}" ]]; then
    printf >&2 "%s\n" "$TRACKED_CHANGED"
  fi
  if [[ -n "${UNTRACKED_CHANGED}" ]]; then
    printf >&2 "%s\n" "$UNTRACKED_CHANGED"
  fi
  exit 1
}

main "$@"
