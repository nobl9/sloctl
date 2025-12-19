# Get all alerts triggered (max 1000).
sloctl get alert -A

# Get only active (not resolved yet) alerts.
sloctl get alert --triggered -A

# Get a specific alert by the alert ID.
sloctl get alert ce1a2a10-d74d-477f-b574-b278ee54e02b -A

# Get alerts related to the reportsapi service or usersapi service in project prod.
sloctl get alert --service reportsapi --service usersapi -p prod

# Get only resolved alerts for the specific alert policy and SLO in the specified project.
sloctl get alert --resolved --alert-policy slow-burn --slo usersapi-latency -p prod

# Get alerts triggered for the slo usersapi-availability AND objective objective-1 in project prod.
sloctl get alert --slo usersapi-availability --objective objective-1 -p prod

# Get alerts triggered for slo usersapi-latency AND objective objective-1 OR objective-2 in project prod.
sloctl get alert --slo usersapi-latency --objective objective-1 --objective objective-2 -p prod

# Get alerts by a time range.
# We're assuming the current date is 2023-03-23T12:00:00Z:
# - Alerts that were active yesterday:
sloctl get alert --from 2023-03-22T00:00:00Z --to 2023-03-22T23:59:59Z -A
# - Alerts that have been active since yesterday:
sloctl get alert --from 2023-03-22T00:00:00Z -A
# - Alerts that have been active until today:
sloctl get alert --to 2023-03-23T00:00:00Z -A
