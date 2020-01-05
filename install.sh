#!/bin/sh

set -o xtrace
set -o errexit
set -o pipefail

VENDOR=halkeye
DRIVER=cifs

# Assuming the single driver file is located at /$DRIVER inside the DaemonSet image.

driver_dir=$VENDOR${VENDOR:+"~"}${DRIVER}

echo 'Installing driver '$driver_dir'/'$DRIVER

if [ ! -d "/flexmnt/$driver_dir" ]; then
  mkdir "/flexmnt/$driver_dir"
fi


export GOARCH=386
if [[ `uname -m` == "x86_64" ]]; then
  export GOARCH="amd64"
fi
cp "/$DRIVER-linux-$GOARCH" "/flexmnt/$driver_dir/.$DRIVER"
mv -f "/flexmnt/$driver_dir/.$DRIVER" "/flexmnt/$driver_dir/$DRIVER"

chmod +x "/flexmnt/$driver_dir/$DRIVER"

echo '
 /$$                 /$$ /$$                                           /$$        /$$  /$$$$$$
| $$                | $$| $$                                          /$$/       |__/ /$$__  $$
| $$$$$$$   /$$$$$$ | $$| $$   /$$  /$$$$$$  /$$   /$$  /$$$$$$      /$$//$$$$$$$ /$$| $$  \__//$$$$$$$
| $$__  $$ |____  $$| $$| $$  /$$/ /$$__  $$| $$  | $$ /$$__  $$    /$$//$$_____/| $$| $$$$   /$$_____/
| $$  \ $$  /$$$$$$$| $$| $$$$$$/ | $$$$$$$$| $$  | $$| $$$$$$$$   /$$/| $$      | $$| $$_/  |  $$$$$$
| $$  | $$ /$$__  $$| $$| $$_  $$ | $$_____/| $$  | $$| $$_____/  /$$/ | $$      | $$| $$     \____  $$
| $$  | $$|  $$$$$$$| $$| $$ \  $$|  $$$$$$$|  $$$$$$$|  $$$$$$$ /$$/  |  $$$$$$$| $$| $$     /$$$$$$$/
|__/  |__/ \_______/|__/|__/  \__/ \_______/ \____  $$ \_______/|__/    \_______/|__/|__/    |_______/
                                             /$$  | $$
                                            |  $$$$$$/
                                             \______/

Driver has been installed.
Make sure /flexmnt from this container mounts to Kubernetes driver directory.

  k8s 1.8.x
  /usr/libexec/kubernetes/kubelet-plugins/volume/exec/

This path may be different in your system due to kubelet parameter --volume-plugin-dir.

This container can now be stopped and removed.

'

while : ; do
  sleep 3600
done
