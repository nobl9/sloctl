# Update Adjustment Events using a file:
cat <<EOF > ./events.yaml
- eventStart: 2024-10-24T04:07:04Z
  eventEnd: 2024-10-24T05:27:04Z
  slos:
  - project: test-project
    name: sample-slo-1
  update:
    eventStart: 2024-10-24T03:07:04Z
    eventEnd: 2024-10-24T04:27:04Z
- eventStart: 2024-10-25T04:07:04Z
  eventEnd: 2024-10-25T05:27:04Z
  slos:
  - project: test-project
    name: sample-slo-2
  update:
    eventStart: 2024-10-25T03:07:04Z
    eventEnd: 2024-10-25T04:27:04Z
EOF
sloctl budgetadjustments events update --adjustment-name=sample-adjustment-name -f ./events.yaml

# Update Adjustment Events using stdin:
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
