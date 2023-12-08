# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
# @date 2023-12-07

variable "ctool_node" {
  description = "Public node IP for ctool building"
  type        = string
}

variable "ssh_private_key" {}

resource "null_resource" "build_ctool" {

  provisioner "remote-exec" {
    inline = [
      "chmod 755 /tmp/amazonKey.pem",
      "curl -L https://git.io/vQhTU | bash -s -- --version 1.21.4",
      "git clone https://github.com/voedger/voedger",
      "export GOROOT=$HOME/.go",
      "export PATH=$GOROOT/bin:$PATH",
      "echo $GOROOT",
      "echo $PATH",
      "cd $HOME/voedger/cmd/ctool && go build -o ctool"
    ]
  }
  connection {
    type        = "ssh"
    user        = "ubuntu"
    private_key = "${var.ssh_private_key}"
    host        = "${var.ctool_node}"
    port        = 2214
    agent       = false
  }
}

