provider "aws" {
  region = "${var.aws_region}"
}


resource "aws_instance" "openvpn" {
  ami                         = "${data.aws_ami.ubuntu.id}"
  instance_type               = "${var.instance_type}"
  key_name                    = "${var.aws_key_pair_name}"
  subnet_id                   = "${element(module.vpc.public_subnets, count.index)}"
  vpc_security_group_ids      = ["${aws_security_group.openvpn.id}"]
  associate_public_ip_address = true
  user_data                   = "${data.template_file.user_data.rendered}"

  tags {
    Name = "${var.instance_name}"
  }
}

resource "null_resource" "provision_openvpn" {
  triggers {
    public_ip = "${aws_instance.openvpn.public_ip}"
  }

  connection {
    type  = "ssh"
    host  = "${aws_instance.openvpn.public_ip}"
    user  = "${var.ssh_user}"
    port  = "${var.ssh_port}"
    agent = true
  }

  provisioner "remote-exec" {
    inline = [
      # wait for user_data script to finish
      "while [ ! -f /tmp/terraform-openvpn-complete ]; do echo 'Waiting for user_data script to complete...'; sleep 2; done",

      # install openvpn via the kylemanna/openvpn docker image (configurable via variables)
      "sudo docker volume create openvpn-data",
      "sudo docker run -v openvpn-data:/etc/openvpn --rm ${var.docker_image} ovpn_genconfig -u udp://${aws_instance.openvpn.public_dns}:1194",
      "yes 'yes' | sudo docker run -v openvpn-data:/etc/openvpn --rm -i ${var.docker_image} ovpn_initpki nopass",

      # start service which was inject by user_data script
      "sudo systemctl start docker.openvpn.service",
      "sudo systemctl enable docker.openvpn.service",

      # generate ovpn file
      "sudo docker run -v openvpn-data:/etc/openvpn --rm -it ${var.docker_image} easyrsa build-client-full ${var.client_name} nopass",
      "sudo docker run -v openvpn-data:/etc/openvpn --rm ${var.docker_image} ovpn_getclient ${var.client_name} > ~/${var.client_name}.ovpn",
    ]
  }

  provisioner "local-exec" {
    command = "ssh-keygen -R ${aws_instance.openvpn.public_ip} && ssh-keyscan -H ${aws_instance.openvpn.public_ip} >> ~/.ssh/known_hosts"
  }

  provisioner "local-exec" {
    # copy the ovpn file back to home directory, which you can then import into networkmanager, or install to /etc/openvpn
    command = "scp ${var.ssh_user}@${aws_instance.openvpn.public_ip}:~/${var.client_name}.ovpn ${var.client_name}.ovpn"
  }
}
