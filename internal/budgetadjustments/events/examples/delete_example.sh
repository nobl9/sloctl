# Delete Adjustment Evens using a file
sloctl budgetadjustments events delete --adjustment-name=sample-adjustment-name -f /path/to/file/with/events

# Delete Adjustment Event using stdin:
echo '
- eventStart: 2024-10-25T04:07:04Z
  eventEnd: 2024-10-25T05:27:04Z
  slos:
  - project: test-project
    name: sample-slo-10
' | sloctl budgetadjustments events delete --adjustment-name=sample-adjustment-name -f -
