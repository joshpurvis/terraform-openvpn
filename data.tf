data "aws_ami" "ubuntu" {
    most_recent = true

    filter {
        name   = "name"
        values = ["ubuntu/images/hvm-ssd/ubuntu-xenial-16.04-amd64-server-*"]
    }

    filter {
        name   = "virtualization-type"
        values = ["hvm"]
    }

    owners = ["099720109477"]  # Canonical
}

data "template_file" "openvpn_service" {
  template = "${file("${path.module}/openvpn.service")}"

  vars {
    docker_image = "${var.docker_image}"
  }
}