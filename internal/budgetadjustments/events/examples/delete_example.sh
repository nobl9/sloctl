# Delete Adjustment Events using a file:
cat <<EOF > ./events.yaml
- eventStart: 2024-10-24T04:07:04Z
  eventEnd: 2024-10-24T05:27:04Z
  slos:
  - project: test-project
    name: sample-slo-1
- eventStart: 2024-10-25T04:07:04Z
  eventEnd: 2024-10-25T05:27:04Z
  slos:
  - project: test-project
    name: sample-slo-2
EOF
sloctl budgetadjustments events delete --adjustment-name=sample-adjustment-name -f ./events.yaml

# Delete Adjustment Events using stdin:
echo '
- eventStart: 2024-10-25T04:07:04Z
  eventEnd: 2024-10-25T05:27:04Z
  slos:
  - project: test-project
    name: sample-slo-10
' | sloctl budgetadjustments events delete --adjustment-name=sample-adjustment-name -f -
