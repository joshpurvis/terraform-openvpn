resource "aws_instance" "openvpn" {
  ami                         = "${data.aws_ami.ubuntu.id}"
  instance_type               = "${var.instance_type}"
  key_name                    = "${var.aws_key_pair_name}"
  subnet_id                   = "${element(module.vpc.public_subnets, count.index)}"
  vpc_security_group_ids      = ["${aws_security_group.openvpn.id}"]
  associate_public_ip_address = true

  tags {
    Name = "openvpn"
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

  provisioner "file" {
    content     = "${data.template_file.openvpn_service.rendered}"
    destination = "~/docker.openvpn.service"
  }

  provisioner "remote-exec" {
    inline = [
      # HACK: copy file to permissioned area for provisioner.file per:
      # https://github.com/hashicorp/terraform/issues/8238#issuecomment-240285760
      "sudo cp ~/docker.openvpn.service /etc/systemd/system/docker.openvpn.service",

      # wait for cloudinit to release dpkg/apt lock
      "sleep 10 && while sudo fuser /var/lib/dpkg/lock /var/lib/apt/lists/lock /var/cache/apt/archives/lock >/dev/null 2>&1; do echo 'Waiting for release of dpkg/apt locks'; sleep 5; done;",

      # install docker
      "sudo apt-get install -y apt-transport-https ca-certificates",
      "sudo apt-key adv --keyserver hkp://ha.pool.sks-keyservers.net:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D",
      "echo \"deb https://apt.dockerproject.org/repo ubuntu-xenial main\" | sudo tee /etc/apt/sources.list.d/docker.list",
      "sudo apt-get update",
      "sudo apt-get install -y docker-engine",
      "sudo service docker start",
      "sudo usermod -aG docker $USER",

      # install openvpn via the kylemanna/openvpn docker image (configurable via variables)
      "sudo docker volume create openvpn-data",
      "sudo docker run -v openvpn-data:/etc/openvpn --rm ${var.docker_image} ovpn_genconfig -u udp://${aws_instance.openvpn.public_dns}:1194",
      "yes 'yes' | sudo docker run -v openvpn-data:/etc/openvpn --rm -i ${var.docker_image} ovpn_initpki nopass",

      # configure systemd to manage docker container
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
