variable "aws_region" {
  default = "eu-west-1"
}

provider "aws" {
  profile = "default"
  region  = "${var.aws_region}"
}

resource "aws_key_pair" "deployer" {
  key_name   = "deployer"
  public_key = "${file("/home/josh/.ssh/id_rsa.pub")}"
}

module "openvpn" {
  source            = "../"
  aws_region        = "${var.aws_region}"
  aws_key_pair_name = "${aws_key_pair.deployer.key_name}"

  # optional
  client_name      = "terraform-openvpn-${var.aws_region}"
  docker_image     = "kylemanna/openvpn"
}