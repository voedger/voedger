# Copyright (c) 2023 Sigma-Soft, Ltd.
# @author Aleksei Ponomarev
# @date 2023-12-07

variable "ctool_node" {
  description = "Public node IP for ctool building"
  type        = string
}

variable "ssh_private_key" {}

variable "git_repo_url" {
  description = "URL of the Git repository for ctool"
  type        = string
  default     = "https://github.com/voedger/voedger"
}

variable "ssh_port" {
    description = "SSH port for ctool building"
    type        = number
    default     = 22
}

variable "git_commit_id" {}

resource "null_resource" "build_ctool" {

  provisioner "remote-exec" {
    inline = [
      "echo 'Commit id: ' ${var.git_commit_id}",
      "chmod 755 /tmp/amazonKey.pem",
      "curl -L https://git.io/vQhTU | bash -s -- --version 1.21.4",
      "git clone ${var.git_repo_url}",
      "cd voedger",
      var.git_commit_id != "" ? "git checkout ${var.git_commit_id} && git log -n 1" : ":",
      "export GOROOT=$HOME/.go",
      "export PATH=$GOROOT/bin:$PATH",
      "echo 'export GOROOT=$HOME/.go' >> ~/.profile",
      "echo 'export PATH=$GOROOT/bin:$PATH' >> ~/.profile",
      "echo $GOROOT",
      "echo $PATH",
      "cd $HOME/voedger/cmd/ctool && go build -o ctool && ./ctool version"
    ]
  }
  connection {
    type        = "ssh"
    user        = "ubuntu"
    private_key = var.ssh_private_key
    host        = var.ctool_node
    port        = var.ssh_port
    agent       = false
  }
}

