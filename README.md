# About

This repository contains code that provides a working PXE server (via HTTP, TFTP, DHCP, and/or iPXE) implemented purely in Golang. 

Currently, only Linux(CentOS[6/7/8], RHEL[6/7/8], Debian and Ubuntu) which is boot from pxe is supported. See the issues page for open issues, bugs, and enhancements/improvements.

# Usage

## QuickStart

`pxesrv` uses all three services in combination. Run `pxesrv` with --help or -h to see what command line arguments you can pass. 

The following are configs that can be passed to `pxesrv` when running from the command line:

```yaml
# http server config
http:
  listen_ip: 0.0.0.0
  listemn_port: 80
  rootpath: netboot

# tftp config
tftp:
  tftp_root: netboot
  listen_ip: 0.0.0.0
  listen_port: 69

# dhcp config
dhcp:
  listen_ip: 0.0.0.0
  listen_port: 67
  start_ip: 192.168.1.201
  lease_range: 10
  netmask: 255.255.255.0
  router: 192.168.1.1
  dns_server: 114.114.114.114
  tftp_server: 192.168.1.61
  pxe_file: http://192.168.1.61/menu.ipxe

common:
  #root_path: E:\ProgramData\Go\workspace\pxesrv
  root_path: /usr/local/pxeserver
  next_server: http://192.168.1.61

```

## How to use it

### Requirements

- Redhat based OS (CentOS, Redhat Linux, Fedora...)
- Some packages: rpmbuild, zip, tar, ...

### Build

```bash
git clone https://github.com/DongJeremy/pxesrv
cd pxesrv
sh build
```

### Install

Execute the following command in CentOS7/RHEL7

```bash
rpm -ivh /tmp/dist/pxesrv-0.0.1-1.el7.x86_64.rpm
systemctl start pxesrv.service
```

### Systemd (to start up on boot)

```bash
systemctl enable pxesrv.service
```

### Configure

Edit the configuration file "`/usr/local/pxeserver/pxe.yml`"

then mount the iso file to Specific directory. for example:

```bash
mount /root/CentOS-7-x86_64-Minimal-1908.iso /usr/local/pxeserver/netboot/centos/7 -o loop
```

# License

[MIT](http://opensource.org/licenses/MIT)