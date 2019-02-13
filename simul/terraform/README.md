# Deploy virtual machines on EC2 using Terraform

## TL;DR

- install terraform on your machine: https://www.terraform.io/downloads.html
- configure credentials for your AWS account (same way as you would for your `aws-cli`): https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html
- cd to the `aws/` folder
- edit the `main.tf` file to suit your needs.
	- you need to have one "block" per region. Just copy the existing one and rename all occurences of the region name
	- set, for each region the `count` parameter in the `aws_instance` resource
- run `terraform init`
- run `terraform plan`, check if it looks OK
- run `terraform apply`
- run `terraform destroy` to remove all created resources.

*sidenote*: `terraform apply` accepts a `-parallelism=n` argument, telling it how many resources it can create concurrently.

## Warnings

Terraform will create a `.terraform` folder where it will store stuff. The most important thing will be the Terraform state file.

The terraform state file holds the names or ids of all the created resources, and it is required for deletion.

Best practice mandates that the state file is shared, through S3 and locked through DynamoDB - we will not be doing this here.

This means that the `terraform` config should be applied only from a single workstation. If the state file is lost, the resources will have to be deleted manually.

The state file should _never_ be commited in the repository.
