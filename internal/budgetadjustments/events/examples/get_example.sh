# Get Adjustment Events for 'sample-adjustment-name' from 2024-09-23T00:45:00 UTC to 2024-09-23T20:46:00 UTC.
sloctl budgetadjustments events get --adjustment-name=sample-adjustment-name --from=2024-09-23T00:45:00Z --to=2024-09-23T20:46:00Z


# Get Adjustment Events for 'sample-adjustment-name' from 2024-09-23T00:45:00 UTC to 2024-09-23T20:46:00 UTC
# only for one slo with sloName and project filters.
sloctl budgetadjustments events get \
  --adjustment-name=sample-adjustment-name \
  --from=2024-09-23T00:45:00Z \
  --to=2024-09-23T20:46:00Z \
  --slo-project=sample-project-name \
  --slo-name=sample-slo-name
