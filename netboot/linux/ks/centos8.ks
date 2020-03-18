#version=RHEL8
ignoredisk --only-use=sda
# System bootloader configuration
bootloader --append=" crashkernel=auto" --location=mbr --boot-drive=sda
# Partition clearing information
clearpart --all --initlabel --drives=sda
# Reboot after installation
reboot --eject
# Use text mode install
text
# Use network installation
url --url="http://192.168.1.21/centos/8"
# Keyboard layouts
keyboard --vckeymap=us --xlayouts=''
# System language
lang en_US.UTF-8

# Network information
network  --bootproto=dhcp --device=enp0s3
# Root password
rootpw --plaintext password
# System authorization information
authselect
# SELinux configuration
selinux --permissive
# Run the Setup Agent on first boot
firstboot --enable
# Do not configure the X Window System
skipx
# System services
services --enabled="chronyd"
# System timezone
timezone Asia/Shanghai --isUtc
# Disk partitioning information
part /boot --fstype="xfs" --ondisk=sda --size=1024
part biosboot --fstype="biosboot" --ondisk=sda --size=1
part pv.97 --fstype="lvmpv" --ondisk=sda --size 1 --grow
volgroup rhel --pesize=4096 pv.97
logvol swap --fstype="swap" --recommended --name=swap --vgname=rhel
logvol / --fstype="xfs" --grow --size=1 --name=root --vgname=rhel

%packages
@core
chrony
kexec-tools

%end