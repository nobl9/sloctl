# Update Adjustment Evens using a file
sloctl budgetadjustments events update --adjustment-name=sample-adjustment-name -f /path/to/file/with/events

# Update Adjustment Event using stdin:
echo '
- eventStart: 2024-10-25T04:07:04Z
  eventEnd: 2024-10-25T05:27:04Z
  slos:
  - project: test-project
    name: sample-slo-10
  update:
    eventStart: 2024-10-25T03:07:04Z
    eventEnd: 2024-10-25T04:27:04Z
' | sloctl budgetadjustments events update --adjustment-name=sample-adjustment-name -f -
