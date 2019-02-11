### Copy this block for each region, and replace region name everywhere.
# Set the `count` parameter in the `aws_instance` resource to your liking.
# Also, HCL is a declarative, not a programming language.

## Ireland

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

  ingress {
    from_port = 0
    to_port = 0
    protocol = -1
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port = 22
    to_port = 22
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
      from_port   = 0
      to_port     = 65535
      protocol    = "udp"
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
  count = "${var.number_of_instances_per_region}"
  provider = "aws.eu-west-1"
  ami = "${var.ami["eu-west-1"]}"
  instance_type = "${var.instance_type}"
  security_groups = ["${aws_security_group.eu-west-1.name}"]
  key_name = "SIMKEY"
  tags = {
   Name = "R&D"
 }
}


### End of block

## Mumbai

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

  ingress {
    from_port = 0
    to_port = 0
    protocol = -1
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
      from_port   = 0
      to_port     = 65535
      protocol    = "udp"
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
  count = "${var.number_of_instances_per_region}"
  provider = "aws.ap-south-1"
  ami = "${var.ami["ap-south-1"]}"
  instance_type = "${var.instance_type}"
  security_groups = ["${aws_security_group.ap-south-1.name}"]
  key_name = "SIMKEY"
  tags = {
   Name = "R&D"
 }
}
### End of block


## Virginia

provider "aws" {
  alias = "us-east-1"
  region = "us-east-1"
}

resource "aws_security_group" "us-east-1" {
  provider = "aws.us-east-1"
  name = "ssh_ingress"
  ingress {
    from_port = 22
    to_port = 22
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
      from_port   = 0
      to_port     = 65535
      protocol    = "udp"
      cidr_blocks = ["0.0.0.0/0"]
    }

  ingress {
      from_port = 0
      to_port = 0
      protocol = -1
      cidr_blocks = ["0.0.0.0/0"]
    }

  egress {
    from_port = 0
    to_port = 0
    protocol = -1
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_key_pair" "us-east-1" {
  provider = "aws.us-east-1"
  key_name = "SIMKEY"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCVoG80WqVW1Xj8f8HCiKNXKbT2KwTE4IaTBPjlHIsV9qm0x2yfi5M4jD7riULTxza1D2/RcuJCLtlZdi1PhGN3QAAhYBnLdxl1BECc825xKOJ+WWumUoyctCPSVEnTjW8dRPlVcswYqS0S0BV1/hn0XHQ7GhNqzF2FPh7EZgBid0jotsyIm5k/IX8nIWuRJN8n2K6q0dAgdLL+Z8juo3DMRBLQ81tRuo+oZNRf6aolflD4Te/1GV06s/Rrl/Js59szqCUUdYt7ngULRaZNq0HRsO+Qqdo2dhW4pkyoAcOTFjXd3uPbpk3BXIHTk6yv7Xc+aBV/jFCupnGoEKt3b1P9"
}

resource "aws_instance" "us-east-1" {
  count = "${var.number_of_instances_per_region}"
  provider = "aws.us-east-1"
  ami = "${var.ami["us-east-1"]}"
  instance_type = "${var.instance_type}"
  security_groups = ["${aws_security_group.us-east-1.name}"]
  key_name = "SIMKEY"
  tags = {
   Name = "R&D"
 }
}
### End of block

## Seoul

provider "aws" {
  alias = "ap-northeast-2"
  region = "ap-northeast-2"
}

resource "aws_security_group" "ap-northeast-2" {
  provider = "aws.ap-northeast-2"
  name = "ssh_ingress"
  ingress {
    from_port = 22
    to_port = 22
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
      from_port   = 0
      to_port     = 65535
      protocol    = "udp"
      cidr_blocks = ["0.0.0.0/0"]
    }

  ingress {
      from_port = 0
      to_port = 0
      protocol = -1
      cidr_blocks = ["0.0.0.0/0"]
    }

  egress {
    from_port = 0
    to_port = 0
    protocol = -1
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_key_pair" "ap-northeast-2" {
  provider = "aws.ap-northeast-2"
  key_name = "SIMKEY"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCVoG80WqVW1Xj8f8HCiKNXKbT2KwTE4IaTBPjlHIsV9qm0x2yfi5M4jD7riULTxza1D2/RcuJCLtlZdi1PhGN3QAAhYBnLdxl1BECc825xKOJ+WWumUoyctCPSVEnTjW8dRPlVcswYqS0S0BV1/hn0XHQ7GhNqzF2FPh7EZgBid0jotsyIm5k/IX8nIWuRJN8n2K6q0dAgdLL+Z8juo3DMRBLQ81tRuo+oZNRf6aolflD4Te/1GV06s/Rrl/Js59szqCUUdYt7ngULRaZNq0HRsO+Qqdo2dhW4pkyoAcOTFjXd3uPbpk3BXIHTk6yv7Xc+aBV/jFCupnGoEKt3b1P9"
}

resource "aws_instance" "ap-northeast-2" {
  count = "${var.number_of_instances_per_region}"
  provider = "aws.ap-northeast-2"
  ami = "${var.ami["ap-northeast-2"]}"
  instance_type = "${var.instance_type}"
  security_groups = ["${aws_security_group.ap-northeast-2.name}"]
  key_name = "SIMKEY"
  tags = {
   Name = "R&D"
 }
}
### End of block



## Singapore

provider "aws" {
  alias = "ap-southeast-1"
  region = "ap-southeast-1"
}

resource "aws_security_group" "ap-southeast-1" {
  provider = "aws.ap-southeast-1"
  name = "ssh_ingress"
  ingress {
    from_port = 22
    to_port = 22
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
      from_port   = 0
      to_port     = 65535
      protocol    = "udp"
      cidr_blocks = ["0.0.0.0/0"]
    }

  ingress {
      from_port = 0
      to_port = 0
      protocol = -1
      cidr_blocks = ["0.0.0.0/0"]
    }

  egress {
    from_port = 0
    to_port = 0
    protocol = -1
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_key_pair" "ap-southeast-1" {
  provider = "aws.ap-southeast-1"
  key_name = "SIMKEY"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCVoG80WqVW1Xj8f8HCiKNXKbT2KwTE4IaTBPjlHIsV9qm0x2yfi5M4jD7riULTxza1D2/RcuJCLtlZdi1PhGN3QAAhYBnLdxl1BECc825xKOJ+WWumUoyctCPSVEnTjW8dRPlVcswYqS0S0BV1/hn0XHQ7GhNqzF2FPh7EZgBid0jotsyIm5k/IX8nIWuRJN8n2K6q0dAgdLL+Z8juo3DMRBLQ81tRuo+oZNRf6aolflD4Te/1GV06s/Rrl/Js59szqCUUdYt7ngULRaZNq0HRsO+Qqdo2dhW4pkyoAcOTFjXd3uPbpk3BXIHTk6yv7Xc+aBV/jFCupnGoEKt3b1P9"
}

resource "aws_instance" "ap-southeast-1" {
  count = "${var.number_of_instances_per_region}"
  provider = "aws.ap-southeast-1"
  ami = "${var.ami["ap-southeast-1"]}"
  instance_type = "${var.instance_type}"
  security_groups = ["${aws_security_group.ap-southeast-1.name}"]
  key_name = "SIMKEY"
  tags = {
   Name = "R&D"
 }
}
### End of block

## Sydney

provider "aws" {
  alias = "ap-southeast-2"
  region = "ap-southeast-2"
}

resource "aws_security_group" "ap-southeast-2" {
  provider = "aws.ap-southeast-2"
  name = "ssh_ingress"
  ingress {
    from_port = 22
    to_port = 22
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
      from_port   = 0
      to_port     = 65535
      protocol    = "udp"
      cidr_blocks = ["0.0.0.0/0"]
    }

  ingress {
      from_port = 0
      to_port = 0
      protocol = -1
      cidr_blocks = ["0.0.0.0/0"]
    }

  egress {
    from_port = 0
    to_port = 0
    protocol = -1
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_key_pair" "ap-southeast-2" {
  provider = "aws.ap-southeast-2"
  key_name = "SIMKEY"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCVoG80WqVW1Xj8f8HCiKNXKbT2KwTE4IaTBPjlHIsV9qm0x2yfi5M4jD7riULTxza1D2/RcuJCLtlZdi1PhGN3QAAhYBnLdxl1BECc825xKOJ+WWumUoyctCPSVEnTjW8dRPlVcswYqS0S0BV1/hn0XHQ7GhNqzF2FPh7EZgBid0jotsyIm5k/IX8nIWuRJN8n2K6q0dAgdLL+Z8juo3DMRBLQ81tRuo+oZNRf6aolflD4Te/1GV06s/Rrl/Js59szqCUUdYt7ngULRaZNq0HRsO+Qqdo2dhW4pkyoAcOTFjXd3uPbpk3BXIHTk6yv7Xc+aBV/jFCupnGoEKt3b1P9"
}

resource "aws_instance" "ap-southeast-2" {
  count = "${var.number_of_instances_per_region}"
  provider = "aws.ap-southeast-2"
  ami = "${var.ami["ap-southeast-2"]}"
  instance_type = "${var.instance_type}"
  security_groups = ["${aws_security_group.ap-southeast-2.name}"]
  key_name = "SIMKEY"
  tags = {
   Name = "R&D"
 }
}
### End of block

## Tokyo

provider "aws" {
  alias = "ap-northeast-1"
  region = "ap-northeast-1"
}

resource "aws_security_group" "ap-northeast-1" {
  provider = "aws.ap-northeast-1"
  name = "ssh_ingress"
  ingress {
    from_port = 22
    to_port = 22
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
      from_port   = 0
      to_port     = 65535
      protocol    = "udp"
      cidr_blocks = ["0.0.0.0/0"]
    }

  ingress {
      from_port = 0
      to_port = 0
      protocol = -1
      cidr_blocks = ["0.0.0.0/0"]
    }

  egress {
    from_port = 0
    to_port = 0
    protocol = -1
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_key_pair" "ap-northeast-1" {
  provider = "aws.ap-northeast-1"
  key_name = "SIMKEY"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCVoG80WqVW1Xj8f8HCiKNXKbT2KwTE4IaTBPjlHIsV9qm0x2yfi5M4jD7riULTxza1D2/RcuJCLtlZdi1PhGN3QAAhYBnLdxl1BECc825xKOJ+WWumUoyctCPSVEnTjW8dRPlVcswYqS0S0BV1/hn0XHQ7GhNqzF2FPh7EZgBid0jotsyIm5k/IX8nIWuRJN8n2K6q0dAgdLL+Z8juo3DMRBLQ81tRuo+oZNRf6aolflD4Te/1GV06s/Rrl/Js59szqCUUdYt7ngULRaZNq0HRsO+Qqdo2dhW4pkyoAcOTFjXd3uPbpk3BXIHTk6yv7Xc+aBV/jFCupnGoEKt3b1P9"
}

resource "aws_instance" "ap-northeast-1" {
  count = "${var.number_of_instances_per_region}"
  provider = "aws.ap-northeast-1"
  ami = "${var.ami["ap-northeast-1"]}"
  instance_type = "${var.instance_type}"
  security_groups = ["${aws_security_group.ap-northeast-1.name}"]
  key_name = "SIMKEY"
  tags = {
   Name = "R&D"
 }
}
### End of block


## Canada

provider "aws" {
  alias = "ca-central-1"
  region = "ca-central-1"
}

resource "aws_security_group" "ca-central-1" {
  provider = "aws.ca-central-1"
  name = "ssh_ingress"
  ingress {
    from_port = 22
    to_port = 22
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
      from_port   = 0
      to_port     = 65535
      protocol    = "udp"
      cidr_blocks = ["0.0.0.0/0"]
    }

  ingress {
      from_port = 0
      to_port = 0
      protocol = -1
      cidr_blocks = ["0.0.0.0/0"]
    }

  egress {
    from_port = 0
    to_port = 0
    protocol = -1
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_key_pair" "ca-central-1" {
  provider = "aws.ca-central-1"
  key_name = "SIMKEY"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCVoG80WqVW1Xj8f8HCiKNXKbT2KwTE4IaTBPjlHIsV9qm0x2yfi5M4jD7riULTxza1D2/RcuJCLtlZdi1PhGN3QAAhYBnLdxl1BECc825xKOJ+WWumUoyctCPSVEnTjW8dRPlVcswYqS0S0BV1/hn0XHQ7GhNqzF2FPh7EZgBid0jotsyIm5k/IX8nIWuRJN8n2K6q0dAgdLL+Z8juo3DMRBLQ81tRuo+oZNRf6aolflD4Te/1GV06s/Rrl/Js59szqCUUdYt7ngULRaZNq0HRsO+Qqdo2dhW4pkyoAcOTFjXd3uPbpk3BXIHTk6yv7Xc+aBV/jFCupnGoEKt3b1P9"
}

resource "aws_instance" "ca-central-1" {
  count = "${var.number_of_instances_per_region}"
  provider = "aws.ca-central-1"
  ami = "${var.ami["ca-central-1"]}"
  instance_type = "${var.instance_type}"
  security_groups = ["${aws_security_group.ca-central-1.name}"]
  key_name = "SIMKEY"
  tags = {
   Name = "R&D"
 }
}
### End of block


## Frankfurt

provider "aws" {
  alias = "eu-central-1"
  region = "eu-central-1"
}

resource "aws_security_group" "eu-central-1" {
  provider = "aws.eu-central-1"
  name = "ssh_ingress"
  ingress {
    from_port = 22
    to_port = 22
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
      from_port   = 0
      to_port     = 65535
      protocol    = "udp"
      cidr_blocks = ["0.0.0.0/0"]
    }

    ingress {
      from_port = 0
      to_port = 0
      protocol = -1
      cidr_blocks = ["0.0.0.0/0"]
    }

  egress {
    from_port = 0
    to_port = 0
    protocol = -1
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_key_pair" "eu-central-1" {
  provider = "aws.eu-central-1"
  key_name = "SIMKEY"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCVoG80WqVW1Xj8f8HCiKNXKbT2KwTE4IaTBPjlHIsV9qm0x2yfi5M4jD7riULTxza1D2/RcuJCLtlZdi1PhGN3QAAhYBnLdxl1BECc825xKOJ+WWumUoyctCPSVEnTjW8dRPlVcswYqS0S0BV1/hn0XHQ7GhNqzF2FPh7EZgBid0jotsyIm5k/IX8nIWuRJN8n2K6q0dAgdLL+Z8juo3DMRBLQ81tRuo+oZNRf6aolflD4Te/1GV06s/Rrl/Js59szqCUUdYt7ngULRaZNq0HRsO+Qqdo2dhW4pkyoAcOTFjXd3uPbpk3BXIHTk6yv7Xc+aBV/jFCupnGoEKt3b1P9"
}

resource "aws_instance" "eu-central-1" {
  count = "${var.number_of_instances_per_region}"
  provider = "aws.eu-central-1"
  ami = "${var.ami["eu-central-1"]}"
  instance_type = "${var.instance_type}"
  security_groups = ["${aws_security_group.eu-central-1.name}"]
  key_name = "SIMKEY"
  tags = {
   Name = "R&D"
 }
}
### End of block


## London

provider "aws" {
  alias = "eu-west-2"
  region = "eu-west-2"
}

resource "aws_security_group" "eu-west-2" {
  provider = "aws.eu-west-2"
  name = "ssh_ingress"
  ingress {
    from_port = 22
    to_port = 22
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
      from_port   = 0
      to_port     = 65535
      protocol    = "udp"
      cidr_blocks = ["0.0.0.0/0"]
    }

    ingress {
      from_port = 0
      to_port = 0
      protocol = -1
      cidr_blocks = ["0.0.0.0/0"]
    }

  egress {
    from_port = 0
    to_port = 0
    protocol = -1
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_key_pair" "eu-west-2" {
  provider = "aws.eu-west-2"
  key_name = "SIMKEY"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCVoG80WqVW1Xj8f8HCiKNXKbT2KwTE4IaTBPjlHIsV9qm0x2yfi5M4jD7riULTxza1D2/RcuJCLtlZdi1PhGN3QAAhYBnLdxl1BECc825xKOJ+WWumUoyctCPSVEnTjW8dRPlVcswYqS0S0BV1/hn0XHQ7GhNqzF2FPh7EZgBid0jotsyIm5k/IX8nIWuRJN8n2K6q0dAgdLL+Z8juo3DMRBLQ81tRuo+oZNRf6aolflD4Te/1GV06s/Rrl/Js59szqCUUdYt7ngULRaZNq0HRsO+Qqdo2dhW4pkyoAcOTFjXd3uPbpk3BXIHTk6yv7Xc+aBV/jFCupnGoEKt3b1P9"
}

resource "aws_instance" "eu-west-2" {
  count = "${var.number_of_instances_per_region}"
  provider = "aws.eu-west-2"
  ami = "${var.ami["eu-west-2"]}"
  instance_type = "${var.instance_type}"
  security_groups = ["${aws_security_group.eu-west-2.name}"]
  key_name = "SIMKEY"
  tags = {
   Name = "R&D"
 }
}
### End of block
