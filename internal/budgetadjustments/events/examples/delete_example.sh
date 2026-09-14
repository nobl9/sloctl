# Define one event to delete.
cat <<'EOF' >./events.yaml
- eventStart: 2025-04-07T01:00:00Z
  eventEnd: 2025-04-07T02:00:00Z
  slos:
    - project: my-project
      name: my-slo
EOF
sloctl budgetadjustments events delete \
  --adjustment-name maintenance \
  --file ./events.yaml

# Read the same definition from standard input.
sloctl budgetadjustments events delete \
  --adjustment-name maintenance \
  --file - <./events.yaml
