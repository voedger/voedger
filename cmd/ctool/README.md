# ctool

Deploy [Heeus Community Edition (CE)](https://github.com/heeus/heeus-design#community-edition-ce) and [Heeus Community Edition (CE)](https://github.com/heeus/heeus-design#community-edition-ce) clusters
         
## Quick start

**Prerequisites**
- "Clean"
- The same user
- The same SSH key

### Deploy CE

**Prerequisites**
- Configure 1 clean  Ubunty??? server
  - The following address will be used as example: 5.255.255.56 
- Admin user SSH key: adm.key


**Deploy a CE cluster on a remote host**

    $ ctool init CE 5.255.255.55
    $ ctool apply ./adm.key

**Deploy a CE cluster on a localhost**

    $ ctool init CE
    $ ctool apply [--force]

 `--force` - forced reinstallation of the required environment

### Deploy SE

**Prerequisites**
- Configure 5 clean  Ubunty??? servers
  - The following addresses will be used as example: 5.255.255.56 5.255.255.57 5.255.255.58 5.255.255.59 5.255.255.60 
- Admin user name: adm
- Admin user SSH key: adm.key

**Deploy a SE cluster**

    $ ctool init SE 5.255.255.56 5.255.255.57 5.255.255.58 5.255.255.59 5.255.255.60 
    $ ctool apply [--force] adm ./adm.key

`--force` - forced reinstallation of the required environment

**Deploy a stritched SE cluster**

    $ ctool init SE 5.255.255.56 5.255.255.57 5.255.255.58 5.255.255.59 5.255.255.60 dc1 dc2 dc3 
    $ ctool apply adm ./adm.key

**Re-apply after error**

    $ ctool apply adm ./adm.key

**Update the cluster version to the ctool version**

    $ ctool upgrade
    $ ctool apply adm ./adm.key

**Validate cluster.json structure**

    $ ctool validate

**Replace a cluster node**

    $ ctool replace 5.255.255.56 5.255.255.60
    $ ctool apply adm ./adm.key

**Get low-level information on cluster objects**

    $ ctool inspect

**The result of the execution of the ctool commands**

As a result of executing the commands: `init`, `repair`, `replace`, `upgrade` or `apply` a file `cluster.json` is created in the current folder and a folder
named YYYY-MM-DD--HH-NN-SS-&lt;commandName&gt; containing a detailed cluster deployment log &lt;`commandName`&gt;`.log`.


    cluster.json - contains data about the cluster configuration and the success indicator
                   deployment of a cluster of this configuration. 

    YYYY-MM-DD--HH-NN-SS-commandName/commandName.log - a detailed log of cluster command operations.



## Use custom compose files

To use custom compose files when executing the deploy command, the following options must be used

--app-compose // CENode/SENode

--db-compose  // DBNode
