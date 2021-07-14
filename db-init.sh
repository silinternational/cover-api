#!/usr/bin/env bash
{
  echo "log_connections=on"
  echo "log_min_messages=warning"
  echo "log_statement=all"
} >> /var/lib/postgresql/data/postgresql.conf
