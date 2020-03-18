# this is generated by pxebuilder.
install
text
url --url=http://192.168.1.21/rhel/6
lang en_US.UTF-8
keyboard us
unsupported_hardware
network --device eth0 --onboot yes --bootproto dhcp
rootpw password
firewall --service=ssh
authconfig --enableshadow --passalgo=sha512
selinux --permissive
timezone --utc Asia/Shanghai
bootloader --location=mbr --driveorder=sda,sdb --append="crashkernel=auto"
clearpart --all --initlabel
zerombr

part /boot --fstype=ext4 --size=200
part pv.202002 --grow --size=1
volgroup VolGroup --pesize=4096 pv.202002
logvol / --fstype=ext4 --name=lv_root --vgname=VolGroup --grow --size=1024
logvol swap --name=lv_swap --vgname=VolGroup --size=1000 --grow --maxsize=3968

reboot

%packages --nobase
@core
%end