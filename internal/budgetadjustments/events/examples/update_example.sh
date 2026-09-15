# Identify one event and provide its replacement timestamps.
cat <<'EOF' >./events.yaml
- eventStart: 2025-04-07T01:00:00Z
  eventEnd: 2025-04-07T02:00:00Z
  slos:
    - project: my-project
      name: my-slo
  update:
    eventStart: 2025-04-07T03:00:00Z
    eventEnd: 2025-04-07T04:00:00Z
EOF
sloctl budgetadjustments events update \
  --adjustment-name maintenance \
  --file ./events.yaml

# Read the same definition from standard input.
sloctl budgetadjustments events update \
  --adjustment-name maintenance \
  --file - <./events.yaml
