#!/usr/bin/env bash

set -eo pipefail

read_with_retry() {
  local query_attempt query_exit query_output
  for query_attempt in {1..3}; do
    query_exit=0
    query_output="$(timeout 15s gh "$@")" || query_exit=$?
    if ((query_exit == 0)); then
      printf '%s\n' "${query_output}"
      return 0
    fi
    echo "GitHub API read failed (attempt ${query_attempt}/3, exit ${query_exit})." >&2
    if ((query_attempt == 3)); then
      return "${query_exit}"
    fi
    sleep 2
  done
}

list_matching_run() {
  read_with_retry run list \
    --repo "${GITHUB_REPOSITORY_OWNER}/${DOCS_REPOSITORY}" \
    --workflow update-sloctl-command-reference.yml \
    --event workflow_dispatch \
    --limit 100 \
    --json conclusion,databaseId,displayTitle,status \
    --jq '
      [.[] | select(.displayTitle == env.EXPECTED_RUN_NAME)]
      | sort_by(.databaseId)
      | last
      | if . == null then ""
        else [.databaseId, .status, (.conclusion // "")] | @tsv
        end
    '
}

view_run() {
  local run_id="$1"
  read_with_retry run view "${run_id}" \
    --repo "${GITHUB_REPOSITORY_OWNER}/${DOCS_REPOSITORY}" \
    --json conclusion,status \
    --jq '[.status, (.conclusion // "")] | @tsv'
}

existing_run="$(list_matching_run)"
IFS=$'\t' read -r downstream_run_id run_status run_conclusion <<< "${existing_run}"
if [[ -n "${downstream_run_id}" && "${run_status}" == "completed" ]]; then
  if [[ "${run_conclusion}" == "success" ]]; then
    echo "Documentation workflow run ${downstream_run_id} already completed successfully."
    exit 0
  fi
  echo "Previous documentation workflow run ${downstream_run_id} completed with ${run_conclusion}; dispatching a new attempt."
  previous_run_id="${downstream_run_id}"
  downstream_run_id=""
elif [[ -n "${downstream_run_id}" ]]; then
  echo "Resuming documentation workflow run ${downstream_run_id}, currently ${run_status}."
else
  previous_run_id=""
fi

if [[ -z "${downstream_run_id}" ]]; then
  if ! timeout 30s gh workflow run update-sloctl-command-reference.yml \
    --repo "${GITHUB_REPOSITORY_OWNER}/${DOCS_REPOSITORY}" \
    --ref production \
    --field tag="${SLOCTL_TAG}" \
    --field sha="${SLOCTL_SHA}" \
    --field source_run_id="${SOURCE_RUN_ID}"; then
    # The request can succeed even if the response is lost.
    echo "Could not confirm documentation workflow dispatch. Checking for an accepted run." >&2
  fi

  discovery_deadline=$((SECONDS + 120))
  while ((SECONDS < discovery_deadline)); do
    list_exit=0
    matching_run="$(list_matching_run)" || list_exit=$?
    if ((list_exit == 0)); then
      IFS=$'\t' read -r downstream_run_id run_status run_conclusion <<< "${matching_run}"
      if [[ -n "${downstream_run_id}" && "${downstream_run_id}" != "${previous_run_id}" ]]; then
        break
      fi
    else
      echo "Could not list documentation runs during discovery (exit ${list_exit})." >&2
    fi
    sleep 2
  done

  if [[ -z "${downstream_run_id}" || "${downstream_run_id}" == "${previous_run_id}" ]]; then
    echo "Could not find the dispatched documentation workflow run." >&2
    exit 1
  fi
fi

watch_deadline=$((SECONDS + 45 * 60))
for watch_attempt in {1..3}; do
  watch_exit=124
  remaining=$((watch_deadline - SECONDS))
  if ((remaining > 0)); then
    watch_exit=0
    timeout "${remaining}s" gh run watch "${downstream_run_id}" \
      --repo "${GITHUB_REPOSITORY_OWNER}/${DOCS_REPOSITORY}" \
      --exit-status \
      --interval 10 || watch_exit=$?
  fi
  if ((watch_exit == 0)); then
    exit 0
  fi

  if ! run_result="$(view_run "${downstream_run_id}")"; then
    echo "Could not determine the state of documentation workflow run ${downstream_run_id}. Rerun Sync sloctl docs to resume monitoring." >&2
    exit 1
  fi
  IFS=$'\t' read -r run_status run_conclusion <<< "${run_result}"
  if [[ "${run_status}" == "completed" ]]; then
    if [[ "${run_conclusion}" == "success" ]]; then
      exit 0
    fi
    echo "Documentation workflow run ${downstream_run_id} completed with ${run_conclusion} (watch exit ${watch_exit})." >&2
    exit 1
  fi

  if ((watch_exit == 124 || SECONDS >= watch_deadline)); then
    break
  fi
  if ((watch_attempt == 3)); then
    echo "Monitoring failed after three attempts (exit ${watch_exit}). Rerun Sync sloctl docs to resume run ${downstream_run_id}." >&2
    exit 1
  fi
  echo "Monitoring stopped with exit ${watch_exit}. Retrying run ${downstream_run_id} (attempt $((watch_attempt + 1))/3)." >&2
  sleep 2
done

echo "Documentation workflow run ${downstream_run_id} exceeded 45 minutes while ${run_status}; cancelling it." >&2
cancel_exit=0
timeout 30s gh run cancel "${downstream_run_id}" \
  --repo "${GITHUB_REPOSITORY_OWNER}/${DOCS_REPOSITORY}" || cancel_exit=$?
if ((cancel_exit != 0)); then
  echo "Could not request cancellation for run ${downstream_run_id} (exit ${cancel_exit})." >&2
fi

cancel_deadline=$((SECONDS + 180))
while ((SECONDS < cancel_deadline)); do
  if ! run_result="$(view_run "${downstream_run_id}")"; then
    sleep 2
    continue
  fi
  IFS=$'\t' read -r run_status run_conclusion <<< "${run_result}"
  if [[ "${run_status}" == "completed" ]]; then
    break
  fi
  sleep 2
done

if [[ "${run_conclusion}" == "success" ]]; then
  echo "Documentation workflow run ${downstream_run_id} completed successfully during cancellation."
  exit 0
fi
if [[ "${run_status}" != "completed" ]]; then
  echo "Documentation workflow run ${downstream_run_id} did not stop after cancellation." >&2
  exit 1
fi
echo "Documentation workflow run ${downstream_run_id} stopped with ${run_conclusion}." >&2
exit 1
