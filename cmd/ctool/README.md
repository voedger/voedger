# ctool

Deploy [Heeus Community Edition (CE)](https://github.com/heeus/heeus-design#community-edition-ce) and [Heeus Community Edition (CE)](https://github.com/heeus/heeus-design#community-edition-ce) clusters
         
## Quick start

**Prerequisites**
- "Clean"
- The same user
- The same SSH key

### Deploy N1

**Prerequisites**
- Configure 1 clean  Ubunty??? server
  - The following address will be used as example: 5.255.255.56 
- Admin user SSH key: adm.key


**Deploy a N1 cluster on a localhost**

    $ ./ctool init n1 10.0.0.21


### Deploy N5

**Prerequisites**
- Configure 5 clean Ubunty servers
  - The following addresses will be used as example: 5.255.255.56 5.255.255.57 5.255.255.58 5.255.255.59 5.255.255.60 
- Admin user SSH key: adm.key

**Deploy a N5 cluster**

    $ ./ctool init n5 5.255.255.56 5.255.255.57 5.255.255.58 5.255.255.59 5.255.255.60 --ssh-key ./adm.key

### Deploy N3

**Prerequisites**
- Configure 3 clean Ubunty servers
  - The following addresses will be used as example: 5.255.255.56 5.255.255.57 5.255.255.58 5.255.255.59 5.255.255.60 
- Admin user SSH key: adm.key

**Deploy a N3 cluster**

    $ ./ctool init n3 5.255.255.56 5.255.255.57 5.255.255.58 --ssh-key ./adm.key


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

