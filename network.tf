module "vpc" {
  source = "terraform-aws-modules/vpc/aws"

  name = "openvpn"
  cidr = "10.1.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support = true

  azs             = ["eu-west-1a", "eu-west-1b", "eu-west-1c"]
  public_subnets = ["10.1.1.0/24", "10.1.2.0/24", "10.1.3.0/24"]

  tags = {
    Name = "openvpn"
  }
}