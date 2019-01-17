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

variable "ami" {
  type = "map"
  description = "AMI / Region Map"
  default = {
    ap-northeast-1 = "ami-016ad6443b4a3d960"
    ap-northeast-2 = "ami-0befa04e7b1ba50f9"
    ap-northeast-3 = "ami-0afdc600eb90f06c8"
    ap-south-1 = "ami-04611067ce944d00f"
    ap-southeast-1 = "ami-00cbdef8d2acf44a7"
    ap-southeast-2 = "ami-0fbfb4926256a1f1e"
    ca-central-1 = "ami-08f0313c2834a2ff7"
    eu-central-1 = "ami-0f0debf49705e047c"
    eu-west-1 = "ami-06e710681e5ee07aa"
    eu-west-2 = "ami-096629e5eb19568cc"
    eu-west-3 = "ami-044e19acaba1ddac8"
    sa-east-1 = "ami-088018633b4291710"
    us-east-1 = "ami-02cc5b6cf82d354da"
    us-east-2 = "ami-0ec1948d5caef658a"
    us-west-1 = "ami-0c3ca2c6f4edb5546"
    us-west-2 = "ami-0b5913cdbba67598e"
    cn-north-1 = "ami-0273771427032a4e6"
    cn-northwest-1 = "ami-0ff05dc898c27df83"
  }
}
