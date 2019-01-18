### Copy this block for each region, and replace region name everywhere.
# Set the `count` parameter in the `aws_instance` resource to your liking.
# Also, HCL is a declarative, not a programming language.
provider "aws" {
  alias = "eu-west-1"
  region = "eu-west-1"
}

resource "aws_security_group" "eu-west-1" {
  provider = "aws.eu-west-1"
  name = "ssh_ingress"
  ingress {
    from_port = 22
    to_port = 22
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port = 0
    to_port = 0
    protocol = -1
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_key_pair" "eu-west-1" {
  provider = "aws.eu-west-1"
  key_name = "SIMKEY"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCVoG80WqVW1Xj8f8HCiKNXKbT2KwTE4IaTBPjlHIsV9qm0x2yfi5M4jD7riULTxza1D2/RcuJCLtlZdi1PhGN3QAAhYBnLdxl1BECc825xKOJ+WWumUoyctCPSVEnTjW8dRPlVcswYqS0S0BV1/hn0XHQ7GhNqzF2FPh7EZgBid0jotsyIm5k/IX8nIWuRJN8n2K6q0dAgdLL+Z8juo3DMRBLQ81tRuo+oZNRf6aolflD4Te/1GV06s/Rrl/Js59szqCUUdYt7ngULRaZNq0HRsO+Qqdo2dhW4pkyoAcOTFjXd3uPbpk3BXIHTk6yv7Xc+aBV/jFCupnGoEKt3b1P9"
}

resource "aws_instance" "eu-west-1" {
  count = 1
  provider = "aws.eu-west-1"
  ami = "${var.ami["eu-west-1"]}"
  instance_type = "${var.instance_type}"
  security_groups = ["${aws_security_group.eu-west-1.name}"]
  key_name = "SIMKEY"
}
### End of block

provider "aws" {
  alias = "ap-south-1"
  region = "ap-south-1"
}

resource "aws_security_group" "ap-south-1" {
  provider = "aws.ap-south-1"
  name = "ssh_ingress"
  ingress {
    from_port = 22
    to_port = 22
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port = 0
    to_port = 0
    protocol = -1
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_key_pair" "ap-south-1" {
  provider = "aws.ap-south-1"
  key_name = "SIMKEY"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCVoG80WqVW1Xj8f8HCiKNXKbT2KwTE4IaTBPjlHIsV9qm0x2yfi5M4jD7riULTxza1D2/RcuJCLtlZdi1PhGN3QAAhYBnLdxl1BECc825xKOJ+WWumUoyctCPSVEnTjW8dRPlVcswYqS0S0BV1/hn0XHQ7GhNqzF2FPh7EZgBid0jotsyIm5k/IX8nIWuRJN8n2K6q0dAgdLL+Z8juo3DMRBLQ81tRuo+oZNRf6aolflD4Te/1GV06s/Rrl/Js59szqCUUdYt7ngULRaZNq0HRsO+Qqdo2dhW4pkyoAcOTFjXd3uPbpk3BXIHTk6yv7Xc+aBV/jFCupnGoEKt3b1P9"
}

resource "aws_instance" "ap-south-1" {
  count = 1
  provider = "aws.ap-south-1"
  ami = "${var.ami["ap-south-1"]}"
  instance_type = "${var.instance_type}"
  security_groups = ["${aws_security_group.ap-south-1.name}"]
  key_name = "SIMKEY"
}

### End of block
