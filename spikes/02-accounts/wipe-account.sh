#!/bin/bash
# Delete everything the spike creates in ONE account via the AWS API (single aws-cli container, clean env, Floci only).
# usage: wipe-account.sh <account-id>
a=$1
docker run --rm --network host --entrypoint bash -e AWS_ACCESS_KEY_ID=$a -e AWS_SECRET_ACCESS_KEY=test -e AWS_REGION=us-east-1 \
  -e AWS_ENDPOINT_URL=http://localhost:4577 -e AWS_CONFIG_FILE=/dev/null -e AWS_SHARED_CREDENTIALS_FILE=/dev/null -e AWS_PAGER= \
  amazon/aws-cli -c '
q() { aws "$@" --output text 2>/dev/null; }
for b in $(q s3api list-buckets --query "Buckets[].Name"); do
  aws s3api list-object-versions --bucket $b --query "[Versions,DeleteMarkers][][].[Key,VersionId]" --output text 2>/dev/null | while read k v; do [ -n "$k" ] && aws s3api delete-object --bucket $b --key "$k" --version-id "$v" >/dev/null; done
  aws s3api delete-bucket --bucket $b
done
for p in $(q ssm describe-parameters --query "Parameters[].Name"); do aws ssm delete-parameter --name $p; done
for s in $(q secretsmanager list-secrets --query "SecretList[].ARN"); do aws secretsmanager delete-secret --secret-id $s --force-delete-without-recovery >/dev/null; done
for t in $(q dynamodb list-tables --query "TableNames"); do aws dynamodb delete-table --table-name $t >/dev/null; done
for r in $(q iam list-roles --query "Roles[].RoleName"); do
  for pa in $(q iam list-attached-role-policies --role-name $r --query "AttachedPolicies[].PolicyArn"); do aws iam detach-role-policy --role-name $r --policy-arn $pa; done
  aws iam delete-role --role-name $r
done
for pa in $(q iam list-policies --scope Local --query "Policies[].Arn"); do aws iam delete-policy --policy-arn $pa; done
for g in $(q ec2 describe-security-groups --query "SecurityGroups[?GroupName!=\`default\`].GroupId"); do aws ec2 delete-security-group --group-id $g; done
for s in $(q ec2 describe-subnets --query "Subnets[?DefaultForAz==\`false\`].SubnetId"); do aws ec2 delete-subnet --subnet-id $s; done
for v in $(q ec2 describe-vpcs --query "Vpcs[?IsDefault==\`false\`].VpcId"); do aws ec2 delete-vpc --vpc-id $v; done
'
