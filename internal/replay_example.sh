# Replay one SLO from the specified time until now.
sloctl replay my-slo --project my-project --from 2025-04-07T00:00:00Z

# Replay SLOs from a local configuration file. Values in the file take
# precedence over flags; flags provide defaults for omitted values.
cat <<'EOF' >./replays.yaml
- slo: first-slo
  project: first-project
  from: 2025-04-07T00:00:00Z
- slo: second-slo
EOF
sloctl replay --file ./replays.yaml \
  --project my-project \
  --from 2025-04-08T00:00:00Z
