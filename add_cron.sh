#/bin/bash

# update crontab to keep buri up-to-date
if ! crontab -l | grep buri; then
    line="*/6 * * * * sudo buri install go -a buri -g no/cantara/gotools > /dev/null"
    (crontab -l; echo "$line" ) | crontab -
else
    echo "crontab already contains update of buri"
fi

# update crontab to keep nerthus-cli up-to-date
if ! crontab -l | grep nerthus-cli; then
    line="*/6 * * * * sudo buri install go -a nerthus-cli -g no/cantara/gotools > /dev/null"
    (crontab -l; echo "$line" ) | crontab -
else
    echo "crontab already contains update of nerthus-cli"
fi
