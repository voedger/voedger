provider "aws" {
  region = "us-east-2"
  shared_credentials_files = ["/mnt/b/Projects/ctool/terraform/credentials"]
}

resource "aws_vpc" "scylla_cluster_vpc" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "scylla-vpc"
  }
}

resource "aws_placement_group" "scylla_placement_group" {
  name = "Scylla cluster group"
  strategy = "cluster"
}

resource "aws_subnet" "scylla_subnet" {
  vpc_id            = aws_vpc.scylla_cluster_vpc.id
  cidr_block        = "10.0.0.0/24"
  availability_zone = "us-east-2a"
  map_public_ip_on_launch = true
  tags = {
    Name = "scylla"
  }
}

resource "aws_network_interface" "node-00" {
  subnet_id   = aws_subnet.scylla_subnet.id
  private_ips = ["10.0.0.11"]
  security_groups = [aws_security_group.scylla_hosts.id]
  tags = {
    Name = "node-00-interface"
  }
}

output "public_ip_node_00" {
  value = aws_instance.node-00.public_ip
}

resource "aws_instance" "node-00" {
  ami = "ami-024e6efaf93d85776"
  instance_type = "i3.large"
  root_block_device {
    volume_size = "30"
    volume_type = "gp2"
  }
  placement_group = aws_placement_group.scylla_placement_group.id
  network_interface {
    network_interface_id = aws_network_interface.node-00.id
    device_index         = 0
  }
  key_name = "amazonKeyED25519"
}

resource "aws_internet_gateway" "gw" { vpc_id = aws_vpc.scylla_cluster_vpc.id }

resource "aws_route_table" "cluster_route" {
  vpc_id = aws_vpc.scylla_cluster_vpc.id
  tags = {
    Name = "Cluster route table"
  }
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.gw.id
  }
}

resource "aws_route_table_association" "private" {
  subnet_id      = aws_subnet.scylla_subnet.id
  route_table_id = aws_route_table.cluster_route.id
}
