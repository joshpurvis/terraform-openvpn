resource "aws_security_group" "openvpn" {
  name        = "openvpn"
  vpc_id      = "${module.vpc.vpc_id}"

  ingress {
    from_port   = "22"
    to_port     = "22"
    protocol    = "tcp"
    cidr_blocks = ["${var.ssh_cidr}"]
  }

  ingress {
    from_port   = "943"
    to_port     = "943"
    protocol    = "tcp"
    cidr_blocks = ["${var.openvpn_cidr}"]
  }

  ingress {
    from_port   = "1194"
    to_port     = "1194"
    protocol    = "udp"
    cidr_blocks = ["${var.openvpn_cidr}"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

}
