# Programming the Cloud

This chapter illustrates the basics of manipulating and using cloud infrastructure. We build two stacks of infrastructure, a Virtual Machine and related infra, and a blob storage account and related infra. Through this code, we can learn about the Azure control plane (Azure Resource Manager) and the Azure Storage blob data plane.

**NOTE:** Please DO NOT handle errors like shown in these examples. The error handling is abbreviated to make the code more concise for readability in the book. Panic is not your friend!

## Tools needed
- Azure CLI

## Getting Started
The following steps will help you get started running the examples. Each of the steps should be run from the root directory of this chapter.

### Generating SSH keys
Lets create an SSH key to use for logging into our Virtual Machine.
```shell
$ mkdir .ssh
$ ssh-keygen -t rsa -b 4096
Generating public/private rsa key pair.
Enter file in which to save the key (/Users/user/.ssh/id_rsa): ./.ssh/id_rsa
Enter passphrase (empty for no passphrase):
Enter same passphrase again:
Your identification has been saved in ./.ssh/id_rsa
Your public key has been saved in ./.ssh/id_rsa.pub
The key fingerprint is:
SHA256:WtZQylkZaSjC24LWDw78wqqz8PJm+1eas54xy6Dk0ws user@computer
The key's randomart image is:
+---[RSA 4096]----+
|   .     .++     |
|    o ...=+      |
| . o + .=.       |
|  = = .  o       |
| o + +  S .      |
|  o o .+.        |
|..Eo. ++         |
|=++o.o==         |
|+B*+o+*o         |
+----[SHA256]-----+
```

### Login to the Azure CLI
Now that we have our SSH key setup, lets login to the Azure CLI.
```shell
az login
```

### Create .env file
The following command will create a .env file which is used to load environment vars in the examples.
```shell
echo -e "AZURE_SUBSCRIPTION_ID=$(az account show --query 'id' -o tsv)\nSSH_PUBLIC_KEY_PATH=./.ssh/id_rsa.pub" >> .env
```

## Run the examples
Each of the examples below when executed will describe what operations are taking place and prompt you to interact with the infrastructure provisioned. Once done playing with the provisioned infra, just press enter in the terminal and the infrastructure will be destroyed.

### Building an Azure Virtual Machine and related infrastructure
This example will build an Azure Resource Group, networking infra, and a Virtual Machine. After the VM is built, you can SSH into the machine and explore. The provisioned Virtual Machine runs the cloud-init provisioning script in `./cloud-init/init.yml` upon creation.
```shell
go run ./cmd/compute/main.go
```

### Building an Azure Storage account, related infra, and storing files
This example will build an Azure Resource Group and a blob Storage Account which we will use to store some local images from the `./blobs` directory in the private cloud blob store. The example will then print signed URIs which will grant limited access to the images stored in the storage account.
```shell
go run ./cmd/storage/main.go
```

