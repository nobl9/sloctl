# Get active and resolved alerts from all projects.
sloctl get alerts --all-projects

# Get only active alerts.
sloctl get alerts --all-projects --triggered --resolved=false

# Get only resolved alerts.
sloctl get alerts --all-projects --resolved --triggered=false

# Match either objective while also requiring the specified SLO.
sloctl get alerts --project my-project --slo my-slo \
  --objective availability --objective latency

# Filter by metric-time range.
sloctl get alerts --all-projects \
  --from=2025-03-22T00:00:00Z \
  --to=2025-03-22T23:59:59Z
