#!/bin/bash
sudo rm -f /usr/local/bin/b0uncer
read -p "Remove ~/.b0uncer data? (y/n) " yn
if [ "$yn" = "y" ]; then rm -rf ~/.b0uncer; fi
echo "B0uncer uninstalled."
