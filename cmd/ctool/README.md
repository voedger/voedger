# ctool

Deploy [Heeus Community Edition (CE)](https://github.com/heeus/heeus-design#community-edition-ce) and [Heeus Community Edition (CE)](https://github.com/heeus/heeus-design#community-edition-ce) clusters
         
## Quick start

**Prerequisites**
- "Clean"
- The same user
- The same SSH key

### Deploy N1 (Community Edition)

**Prerequisites**
- Configure 1 clean  Ubuntu server
  - The following address will be used as example: 5.255.255.56
- Admin user SSH key: adm.key

**Deploy a N1 cluster**

    $ ./ctool init n1 10.0.0.21 --ssh-key ./adm.key

**Deploy a CE cluster (alias for n1)**

    $ ./ctool init CE 10.0.0.21 --ssh-key ./adm.key


### Deploy N5 (Standard Edition)

**Prerequisites**
- Configure 5 clean Ubuntu servers
  - The following addresses will be used as example: 5.255.255.56 5.255.255.57 5.255.255.58 5.255.255.59 5.255.255.60
- Admin user SSH key: adm.key

**Deploy a N5 cluster**

    $ ./ctool init n5 5.255.255.56 5.255.255.57 5.255.255.58 5.255.255.59 5.255.255.60 --ssh-key ./adm.key

**Deploy a SE cluster (alias for n5)**

    $ ./ctool init SE 5.255.255.56 5.255.255.57 5.255.255.58 5.255.255.59 5.255.255.60 --ssh-key ./adm.key

### Deploy N3

**Prerequisites**
- Configure 3 clean Ubuntu servers
  - The following addresses will be used as example: 5.255.255.56 5.255.255.57 5.255.255.58
- Admin user SSH key: adm.key

**Deploy a N3 cluster**

    $ ./ctool init n3 5.255.255.56 5.255.255.57 5.255.255.58 --ssh-key ./adm.key

## Backward Compatibility

For backward compatibility with existing scripts and tests, ctool supports deprecated aliases:

- `CE` - alias for `n1` (Community Edition, single node)
- `SE` - alias for `n5` (Standard Edition, 5 nodes)

These aliases work exactly the same as their modern equivalents:

    # These commands are equivalent:
    $ ./ctool init n1 10.0.0.21 --ssh-key ./adm.key
    $ ./ctool init CE 10.0.0.21 --ssh-key ./adm.key

    # These commands are equivalent:
    $ ./ctool init n5 ip1 ip2 ip3 ip4 ip5 --ssh-key ./adm.key
    $ ./ctool init SE ip1 ip2 ip3 ip4 ip5 --ssh-key ./adm.key

**Repeat after error**

    $ ./ctool repeat --ssh-key ./adm.key

**Upgrade the cluster version to the ctool version**

    $ ./ctool upgrade --ssh-key ./adm.key

**Validate cluster.json structure**

    $ ./ctool validate

**Replace a cluster node**

    $ ./ctool replace 5.255.255.56 5.255.255.60 --ssh-key ./adm.key


**The result of the execution of the ctool commands**

As a result of executing the commands: `init`, `repeat`, `replace` or `upgrade` a file `cluster.json` is created in the current folder and a folder
named YYYY-MM-DD--HH-NN-SS-&lt;commandName&gt; containing a detailed cluster deployment log &lt;`commandName`&gt;`.log`.


    cluster.json - contains data about the cluster configuration and the success indicator
                   deployment of a cluster of this configuration. 

    YYYY-MM-DD--HH-NN-SS-commandName/commandName.log - a detailed log of cluster command operations.

## Environment Variables

ctool uses several environment variables to configure deployment behavior. These can be set in your shell environment or passed to scripts:

### Core Configuration Variables

| Variable | Description | Default | Used By |
|----------|-------------|---------|---------|
| `VOEDGER_SSH_KEY` | Path to SSH private key for server access | `~/.ssh/id_rsa` | All deployment scripts |
| `VOEDGER_NODE_SSH_PORT` | SSH port for server connections | `22` | All deployment scripts |
| `VOEDGER_HTTP_PORT` | HTTP port for Voedger application | `80` (CE), `443` (SE/SE3) | Docker containers |
| `VOEDGER_ACME_DOMAINS` | Comma-separated list of domains for ACME/Let's Encrypt certificates | Empty | SE/SE3 deployments |
| `VOEDGER_CE_NODE` | IP address or hostname of CE node | Required for CE | CE deployment scripts |
| `VOEDGER_EDITION` | Cluster edition type (`CE`, `SE`, `SE3`) | Auto-detected | Monitoring configuration |

### System Variables

| Variable | Description | Default | Used By |
|----------|-------------|---------|---------|
| `SSH_USER` | SSH username for server connections | `$LOGNAME` | All SSH operations |
| `SSH_USER_PASSWORD` | Base64-encoded password for sudo operations | Empty | Passwordless sudo setup |
| `HOME` | User home directory | System default | Docker volume mounts |
| `LOGNAME` | Current user's login name | System default | SSH operations |

### Grafana Configuration Variables

| Variable | Description | Used By |
|----------|-------------|---------|
| `ADMIN_USER` | Grafana admin username | Grafana user management scripts |
| `ADMIN_PASSWORD` | Grafana admin password | Grafana user management scripts |
| `USER_NAME` | Target username for operations | User management scripts |
| `USER_LOGIN` | User login name | User creation scripts |
| `USER_PASSWORD` | User password | User creation scripts |
| `NEW_PASSWORD` | New password for password changes | Password change scripts |
| `DATASOURCE_NAME` | Grafana datasource name | Datasource update scripts |
| `NEW_BASIC_AUTH_USER` | New basic auth username | Datasource update scripts |
| `NEW_BASIC_AUTH_PASSWORD` | New basic auth password | Datasource update scripts |
| `DASHBOARD_UID` | Grafana dashboard UID | Dashboard preference scripts |

### Backup and Maintenance Variables

| Variable | Description | Used By |
|----------|-------------|---------|
| `BACKUP_FOLDER` | Target folder for backups | Backup scripts |
| `CTOOL_PATH` | Path to ctool executable | Cron backup scripts |
| `KEY_PATH` | Path to SSH key for backups | Cron backup scripts |
| `EXPIRE` | Backup expiration time (e.g., "7d", "30d") | Backup cleanup scripts |
| `OUTPUT_FORMAT` | Output format for backup lists ("json" or text) | Backup list scripts |

### Docker and Service Variables

| Variable | Description | Used By |
|----------|-------------|---------|
| `SERVICE_NAME` | Docker service name for operations | Service management scripts |
| `DEVELOPER_MODE` | Developer mode setting (0 or 1) | Docker compose preparation |
| `VERSION_STRING` | Docker version string for installation | Docker installation scripts |

### Network and Host Variables

| Variable | Description | Used By |
|----------|-------------|---------|
| `REMOTE_HOST` | Target remote host for operations | Various utility scripts |
| `LOCAL_FILE` | Local file path for transfers | File transfer scripts |
| `REMOTE_FILE` | Remote file path for transfers | File transfer scripts |
| `FOLDER_PATH` | Target folder path | Folder check scripts |
| `MIN_RAM` | Minimum RAM requirement in MB | Host validation scripts |

### Testing Variables (GitHub Actions)

| Variable | Description | Used By |
|----------|-------------|---------|
| `AWS_ACCESS_KEY_ID` | AWS access key for Terraform | Integration tests |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key for Terraform | Integration tests |
| `TF_VAR_*` | Terraform variables | Integration tests |

### Usage Examples

**Setting environment variables for deployment:**

```bash
# Set SSH configuration
export VOEDGER_SSH_KEY="~/my-key.pem"
export VOEDGER_NODE_SSH_PORT="2222"

# Deploy CE with custom HTTP port
export VOEDGER_HTTP_PORT="8080"
./ctool init CE 10.0.0.21

# Deploy SE with ACME domains
export VOEDGER_ACME_DOMAINS="example.com,api.example.com"
./ctool init SE 10.0.0.21 10.0.0.22 10.0.0.23 10.0.0.24 10.0.0.25
```

**Using environment variables in scripts:**

```bash
# Check if passwordless sudo is configured
export VOEDGER_SSH_KEY="~/admin.key"
./scripts/drafts/ce/check-passwordless-sudo.sh 10.0.0.21

# Set up passwordless sudo with password (base64 encoded)
export SSH_USER_PASSWORD=$(echo "mypassword" | base64)
./scripts/drafts/ce/setup-passwordless-sudo.sh 10.0.0.21

# Configure Grafana admin password
export ADMIN_USER="admin"
export ADMIN_PASSWORD="current_password"
export NEW_PASSWORD="new_secure_password"
./scripts/drafts/ce/grafana-admin-password.sh 10.0.0.21

# Set up backup with custom settings
export BACKUP_FOLDER="/mnt/backup/voedger"
export EXPIRE="30d"
./scripts/drafts/ce/set-cron-backup.sh "0 2 * * *"
```

## SSH key

if you run `ssh-agent` before using the `ctool` utility and add the required ssh key, then you don't need to use the `--ssh-key` flag

    $ eval $(ssh-agent)
    $ ssh-add ./adm.key
    $ ./ctool init CE 5.255.255.55

Otherwise, the commands: `init`, `repeat`, `replace` and `upgrade` must be used with the `--ssh-key` flag

    $ ./ctool init CE 5.255.255.55 --ssh-key ./adm.key

## Use custom compose files

To use custom compose files when executing the deploy command, the following options must be used

--app-compose // CENode/SENode

--db-compose  // DBNode

## Backup / restore database

**Backup of one cluster DBNode**

    $ ./ctool backup node [<node> <target folder>] [flags]

`<node>` - address or name of the cluster node,

`<target folder>` - a folder on the node into which the backup

An example of a `ctool backup node` command

    $ ./ctool backup node db-node-1 ~/backups/test-backup/ --ssh-key ~/adm.key

**Backup database on schedule.**

To install the backup database on all the cluster DBNodes, according to the schedule, the Cron planner needs to execute the command

    $ ./ctool backup cron [<cron event>] [flags]

`<cron event>` - event time in cron format, e.g. `@monthly` or `"* */24 * *"`

Flags:

  `-e, --expire` - expire time for backup (e.g. `7d`, `1m`)

  `--ssh-key` - path to SSH key

If the flag is set `--expire`, then with a successful end of the backup, the old backups created earlier than `--expire` will be removed.

Example of the `ctool backup cron` command

    $ ./ctool backup cron "* */24 * * *" --expire 30d --ssh-key ~/adm.key

The scheduled backup is made at `db-node-1`, `db-node-2` and `db-node-3` nodes in the `/mnt/backup/voedger/` folder. The `/mnt/backup/voedger` folders must be created and the user must have the right to read and record in it.
For the next backup in the `/mnt/backup/voneedger` folder, a folder of the species ` yyyymmddhhnnss-backup` is created


**Printing a list of existing backups**

    $ ./ctool backup list [flags]

Flags:
  
`<--json>` - output in JSON format

`<--ssh-key>` - path to SSH key

Example of the `ctool backup list` command

    $ ./ctool backup list --json --ssh-key ~/adm.key

**Database restore**

    $ ./ctool restore <backup name> [flags]

`<backup name>` - the name of the folder with a backup. If the name of the folder is not absolute, then it is searched on the cluster DBNodes in the `/mnt/backup/voneedger/` folder. To perform the database restore, it is necessary that the folder `<backup name>` exist on all the cluster DBNodes.

Flags:

`--ssh-key` - Path to SSH key

Example of the `ctool restore` command

    $ ./ctool restore 20230624120000-backup --ssh-key ~/adm.key

