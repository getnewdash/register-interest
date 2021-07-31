#!/bin/sh

# Builds and deploys the newdash register-interest application
NEWDASH_EXECUTABLE=/usr/local/bin/register-interest

echo "Building newdash binaries..."
sudo su - newdash -c "/home/newdash/build_newdash.sh"

echo "Checking if binaries built ok..."
if [ ! -x "/home/newdash/go/bin/register-interest" ]; then
  echo Newdash binary did not build correctly
  exit 1
fi

if [ -e $NEWDASH_EXECUTABLE ]; then
  echo "Moving the old binaries out of the way..."
  mv -f $NEWDASH_EXECUTABLE $NEWDASH_EXECUTABLE-old
fi

echo "Moving the new binaries into place..."
mv /home/newdash/go/bin/register-interest $NEWDASH_EXECUTABLE
chown root:root $NEWDASH_EXECUTABLE
