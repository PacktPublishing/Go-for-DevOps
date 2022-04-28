packer {
  required_plugins {
    amazon = {
      version = ">= 0.0.1"
      source  = "github.com/hashicorp/amazon"
    }
  }
}

source "amazon-ebs" "ubuntu" {
  access_key    = "your user's access key"
  secret_key    = "[your secret]"
  ami_name      = "ubuntu-amd64-final"
  instance_type = "t2.micro"
  region        = "us-east-2"
  source_ami_filter {
    filters = {
      name                = "ubuntu/images/*ubuntu-xenial-16.04-amd64-server-*"
      root-device-type    = "ebs"
      virtualization-type = "hvm"
    }
    most_recent = true
    owners      = ["099720109477"]
  }
  ssh_username = "ubuntu"
}

build {
  name = "goBook"
  sources = [
    "source.amazon-ebs.ubuntu"
  ]
  // Install Go 1.17.5
  provisioner "shell" {
    environment_vars = [
      "DEBIAN_FRONTEND=noninteractive",
    ]
    inline = [
      "cd ~",
      "mkdir tmp",
      "cd tmp",
      "wget https://golang.org/dl/go1.17.5.linux-amd64.tar.gz",
      "sudo tar -C /usr/local -xzf go1.17.5.linux-amd64.tar.gz",
      "echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile",
      ". ~/.profile",
      "go version",
      "cd ~/",
      "rm -rf tmp/*",
      "rmdir tmp",
    ]
  }
  // Setup user "agent" with SSH key file
  provisioner "shell" {
    inline = [
      "sudo adduser --disabled-password --gecos '' agent",
    ]
  }
  provisioner "file" {
    source      = "./files/agent.pub"
    destination = "/tmp/agent.pub"
  }
  provisioner "shell" {
    inline = [
      "sudo mkdir /home/agent/.ssh",
      "sudo mv /tmp/agent.pub /home/agent/.ssh/authorized_keys",
      "sudo chown agent:agent /home/agent/.ssh",
      "sudo chown agent:agent /home/agent/.ssh/authorized_keys",
      "sudo chmod 400 .ssh/authorized_keys",
    ]
  }

  // Setup agent binary running with systemd file.
  provisioner "shell" { // This installs dbus-launch
    environment_vars = [
      "DEBIAN_FRONTEND=noninteractive",
    ]
    inline = [
      "sudo apt-get install -y dbus",
      "sudo apt-get install -y dbus-x11",
    ]
  }
  provisioner "file" {
    source      = "./files/agent"
    destination = "/tmp/agent"
  }

  provisioner "shell" {
    inline = [
      "sudo mkdir /home/agent/bin",
      "sudo chown agent:agent /home/agent/bin",
      “sudo chmod ug+rwx /home/agent/bin”,
      "sudo mv /tmp/agent /home/agent/bin/agent",
      "sudo chown agent:agent /home/agent/bin/agent",
      "sudo chmod 0770 /home/agent/bin/agent",
    ]
  }

  provisioner "file" {
    source      = "./files/agent.service"
    destination = "/tmp/agent.service"
  }

  provisioner "shell" {
    inline = [
      "sudo mv /tmp/agent.service /etc/systemd/system/agent.service",
      "sudo systemctl enable agent.service",
      "sudo systemctl daemon-reload",
      "sudo systemctl start agent.service",
      "sleep 10",
      "sudo systemctl is-enabled agent.service",
    ]
  }

  // Setup Goss tool on the image.
  provisioner "shell" {
    inline = [
      "cd ~",
      "mkdir tmp",
      "cd tmp",
      "sudo curl -L https://github.com/aelsabbahy/goss/releases/latest/download/goss-linux-amd64 -o /usr/local/bin/goss",
      "sudo chmod +rx /usr/local/bin/goss",
      "goss -v",
      "cd ~/",
      "rm -rf tmp/*",
      "rmdir tmp",
    ]
  }

  // Copy goss for validating our image onto the image.
  provisioner "file" {
    source      = "./files/goss"
    destination = "/home/ubuntu/goss"
  }

  // Run the Goss tool using our validation files.
  provisioner "goss" {
    retry_timeout = "30s"
    tests = [
      "files/goss/goss.yaml",
      "files/goss/files.yaml",
      "files/goss/dbus.yaml",
      "files/goss/process.yaml",
    ]
  }
}
