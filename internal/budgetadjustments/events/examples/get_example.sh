# Get events for one budget adjustment in a time range.
sloctl budgetadjustments events get \
  --adjustment-name maintenance \
  --from 2025-04-07T00:00:00Z \
  --to 2025-04-08T00:00:00Z

# Filter the events by one SLO and its Project.
sloctl budgetadjustments events get \
  --adjustment-name maintenance \
  --from 2025-04-07T00:00:00Z \
  --to 2025-04-08T00:00:00Z \
  --slo-project my-project \
  --slo-name my-slo
