module "vpc" {
  source = "terraform-aws-modules/vpc/aws"

  name                 = "openvpn"
  cidr                 = "${var.vpc_cidr}"
  enable_dns_hostnames = true
  enable_dns_support   = true

  // first available AZ for now
  azs            = ["${data.aws_availability_zones.available.names[0]}"]
  public_subnets = ["${var.subnet_cidr}"]

  tags = {
    Name = "${var.instance_name}"
  }
}
