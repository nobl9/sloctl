# Set review as reviewed using short note flag
sloctl review set-status reviewed prometheus-latency -p default -n "Target met, 20% error budget remaining"

# Set review as skipped in project inferred from context using note flag
sloctl review set-status skipped prometheus-latency --note "Insufficient data for this period"
