# Set review as reviewed using short note flag
sloctl review set prometheus-latency --status reviewed -p default -n "Target met, 20% error budget remaining"

# Set review as skipped using note flag
sloctl review set prometheus-latency --status skipped -p default --note="Insufficient data for this period"

# Set as pending in project inferred from context
sloctl review set datadog-latency --status pending

# Set as overdue
sloctl review set my-slo --status overdue -p production

# Set as not started
sloctl review set new-slo --status notStarted -p staging
