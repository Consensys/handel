# Will be dealt with later.
# variable "aws_master_region" {
#   type = "string"
#   description = "Region where the master node resides"
#   default = "us-west-2"
# }

# variable "config_bucket_name" {
#   type = "string"
#   description = "describe your variable"
#   default = "pegasys-simulation"
# }

variable "instance_type" {
  type = "string"
  description = "EC2 instance type"
  default = "t2.nano"
}

variable "number_of_instances_per_region" {
  description = "number of EC2 instances per region"
  default = 2
}

variable "ami" {
  type = "map"
  description = "AMI / Region Map"
  default = {
    us-east-1 = "ami-0ac019f4fcb7cb7e6"
    us-west-2 = "ami-0bbe6b35405ecebdb"
    ap-south-1 = "ami-0d773a3b7bb2bb1c1"
    ap-northeast-2 = "ami-06e7b9c5e0c4dd014"
    ap-southeast-1 = "ami-0c5199d385b432989"
    ap-southeast-2 = "ami-07a3bd4944eb120a0"
    ap-northeast-1 = "ami-07ad4b1c3af1ea214"
    ca-central-1 = "ami-0427e8367e3770df1"
    eu-central-1 = "ami-0bdf93799014acdc4"
    eu-west-1 = "ami-00035f41c82244dab"
    eu-west-2 = "ami-0b0a60c0a2bd40612"
  }
}
