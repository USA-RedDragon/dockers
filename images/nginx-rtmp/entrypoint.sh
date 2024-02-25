#!/bin/sh

set -e

envsubst "$(env | sed -e 's/=.*//' -e 's/^/\$/g')" < /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf
exec nginx
