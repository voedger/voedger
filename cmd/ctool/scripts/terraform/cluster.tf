provider "aws" {
  region = "us-east-2"
}

resource "aws_vpc" "cluster_vpc" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "cluster-vpc"
  }
}

resource "aws_placement_group" "cluster_placement_group" {
  name = "Cluster group"
  strategy = "cluster"
}

resource "aws_subnet" "cluster_subnet" {
  vpc_id            = aws_vpc.cluster_vpc.id
  cidr_block        = "10.0.0.0/24"
  availability_zone = "us-east-2a"
  map_public_ip_on_launch = true
  tags = {
    Name = "cluster"
  }
}

resource "aws_network_interface" "node" {
  for_each = { for node in var.included_nodes : node => local.node_configurations[node] }
  subnet_id   = aws_subnet.cluster_subnet.id
  private_ips = [each.value.private_ip]
  security_groups = [aws_security_group.cluster_hosts.id]
  tags = {
    Name = "${each.key}_interface"
  }
}

resource "aws_instance" "node" {
  for_each = aws_network_interface.node
  ami = "ami-0568936c8d2b91c4e"
  instance_type = "i3.large"
  root_block_device {
    volume_size = "30"
    volume_type = "gp2"
  }
  placement_group = aws_placement_group.cluster_placement_group.id
  network_interface {
    network_interface_id = each.value.id
    device_index         = 0
  }
  key_name = "amazonKey"
  tags = {
    Name = each.key
  }
}

resource "aws_route53_record" "app-node-01_instance_record" {
  zone_id = "Z09953832CB14VQPR2ZJG"
  name    = "${var.issue_number}-01.cdci.voedger.io"
  type    = "A"
  ttl     = 60

  records = [aws_instance.node["node_00"].public_ip]
}

module "instance_sshd_provisioners" {
  source           = "./modules/provision-sshd"
  ssh_private_key  = var.ssh_private_key
  ssh_private_ips = [for instance in aws_instance.node : instance.public_ip]

  depends_on = [aws_instance.node]
}

module "instance_ctool_provision" {
  source           = "./modules/provision-ctool"

  ssh_private_key  = var.ssh_private_key
  git_commit_id = var.git_commit_id
  git_repo_url = var.git_repo_url
  ssh_port = var.ssh_port
  ctool_node = aws_instance.node["node_00"].public_ip

  depends_on = [ module.instance_sshd_provisioners ]
}

resource "aws_route53_record" "app-node-02_instance_record" {
  count   = local.node_count > 1 ? 1 : 0
  zone_id = "Z09953832CB14VQPR2ZJG"
  name    = "${var.issue_number}-02.cdci.voedger.io"
  type    = "A"
  ttl     = 60

  records = [aws_instance.node["node_01"].public_ip]
}

output "public_ips" {
  value = [for instance in aws_instance.node : instance.public_ip]
}

output "public_ip_node_00" {
  value = aws_instance.node["node_00"].public_ip
}

output "public_ip_node_02" {
  value = aws_instance.node["node_02"].public_ip
}

output "public_ip_node_03" {
  value = contains(keys(aws_instance.node), "node_03") ? aws_instance.node["node_03"].public_ip : null
}

resource "aws_internet_gateway" "gw" { vpc_id = aws_vpc.cluster_vpc.id }

resource "aws_route_table" "cluster_route" {
  vpc_id = aws_vpc.cluster_vpc.id
  tags = {
    Name = "Cluster route table"
  }
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.gw.id
  }
}

resource "aws_route_table_association" "private" {
  subnet_id      = aws_subnet.cluster_subnet.id
  route_table_id = aws_route_table.cluster_route.id
}
