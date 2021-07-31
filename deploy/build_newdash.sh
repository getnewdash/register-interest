#!/bin/sh

# Builds the Newdash application

cd git_repos/register-interest
git pull
PATH=/usr/local/go/bin:$PATH go install .
