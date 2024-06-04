provider "aws" {
  region = "us-east-2"
}

resource "aws_vpc" "ce_vpc" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "scylla-vpc"
  }
}

resource "aws_placement_group" "ce_placement_group" {
  name = "Voedger CE group"
  strategy = "cluster"
}

resource "aws_subnet" "ce_subnet" {
  vpc_id            = aws_vpc.ce_vpc.id
  cidr_block        = "10.0.0.0/24"
  availability_zone = "us-east-2a"
  map_public_ip_on_launch = true
  tags = {
    Name = "scylla"
  }
}

resource "aws_network_interface" "node_ce" {
  subnet_id   = aws_subnet.ce_subnet.id
  private_ips = ["10.0.0.11"]
  security_groups = [aws_security_group.ce_host.id]
  tags = {
    Name = "node_ce_interface"
  }
}

resource "aws_instance" "node_ce" {
  ami = "ami-0568936c8d2b91c4e"
  instance_type = "i3.large"
  root_block_device {
    volume_size = "10"
    volume_type = "gp2"
  }
  placement_group = aws_placement_group.ce_placement_group.id
  network_interface {
    network_interface_id = aws_network_interface.node_ce.id
    device_index         = 0
  }

  key_name = "amazonKey"

  tags = {
    Name = "node_ce"
  }
}

resource "aws_route53_record" "node_ce_instance_record" {
  zone_id = "Z09953832CB14VQPR2ZJG"
  name    = "${var.issue_number}.cdci.voedger.io"
  type    = "A"
  ttl     = 60

  records = [aws_instance.node_ce.public_ip]
}

module "instance_sshd_provisioners" {
  source           = "./modules/provision-sshd"

  ssh_private_key  = var.ssh_private_key
  ssh_private_ips  = [
      aws_instance.node_ce.public_ip,
    ]

  depends_on = [
      aws_instance.node_ce,
    ]
}

module "instance_ctool_provision" {
  source           = "./modules/provision-ctool"

  ssh_private_key  = var.ssh_private_key
  git_commit_id = var.git_commit_id
  git_repo_url = var.git_repo_url
  ssh_port = var.ssh_port
  ctool_node = aws_instance.node_ce.public_ip

  depends_on = [ module.instance_sshd_provisioners ]
}

output "public_ip_node_ce" {
  value = aws_instance.node_ce.public_ip
}

resource "aws_internet_gateway" "gw" { vpc_id = aws_vpc.ce_vpc.id }

resource "aws_route_table" "cluster_route" {
  vpc_id = aws_vpc.ce_vpc.id
  tags = {
    Name = "Cluster route table"
  }
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.gw.id
  }
}

resource "aws_route_table_association" "private" {
  subnet_id      = aws_subnet.ce_subnet.id
  route_table_id = aws_route_table.cluster_route.id
}
