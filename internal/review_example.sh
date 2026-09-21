# Set "prometheus-latency" SLO (non-default Project) review status as "reviewed" and provide a review note.
sloctl review set-status reviewed prometheus-latency -p non-default -n "Target met, 20% error budget remaining"

# Set "prometheus-latency" SLO (default Project) review status as "skipped" and provide a note explaining the reason for skipping.
sloctl review set-status skipped prometheus-latency --note "Insufficient data for this period"