#!/bin/bash

function installOpenVPNService() {
    sudo tee /etc/systemd/system/docker.openvpn.service > /dev/null << EOF
[Unit]
Description=OpenVPN Container
After=docker.service
Requires=docker.service

[Service]
TimeoutStartSec=0
Restart=always
ExecStartPre=-/usr/bin/docker stop %n
ExecStartPre=-/usr/bin/docker rm %n
ExecStartPre=/usr/bin/docker pull ${docker_image}
ExecStart=/usr/bin/docker run -v openvpn-data:/etc/openvpn --rm -p 1194:1194/udp --cap-add=NET_ADMIN ${docker_image}

[Install]
WantedBy=multi-user.target
EOF

}

function installDocker() {
    sudo apt-get update
    sudo apt-get install -y apt-transport-https ca-certificates
    sudo apt-key adv --keyserver hkp://ha.pool.sks-keyservers.net:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D
    echo "deb https://apt.dockerproject.org/repo ubuntu-xenial main" | sudo tee /etc/apt/sources.list.d/docker.list
    sudo apt-get update
    sudo apt-get install -y docker-engine
    sudo service docker start
    sudo usermod -aG docker ${ssh_user}
}

installDocker
installOpenVPNService

sudo touch /tmp/terraform-openvpn-complete

