[![Moov Banner Logo](https://user-images.githubusercontent.com/20115216/104214617-885b3c80-53ec-11eb-8ce0-9fc745fb5bfc.png)](https://github.com/moov-io)

[![GoDoc](https://godoc.org/github.com/moov-io/frbstatus?status.svg)](https://godoc.org/github.com/moov-io/frbstatus)
[![Build Status](https://github.com/moov-io/frbstatus/workflows/Go/badge.svg)](https://github.com/moov-io/frbstatus/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/moov-io/frbstatus)](https://goreportcard.com/report/github.com/moov-io/frbstatus)
[![Repo Size](https://img.shields.io/github/languages/code-size/moov-io/frbstatus?label=project%20size)](https://github.com/moov-io/frbstatus)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/moov-io/frbstatus/master/LICENSE)
[![Slack Channel](https://slack.moov.io/badge.svg?bg=e01563&fgColor=fffff)](https://slack.moov.io/)
[![GitHub Stars](https://img.shields.io/github/stars/moov-io/frbstatus)](https://github.com/moov-io/frbstatus)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/moov-io/frbstatus)

# FRB Status CLI

A CLI tool to monitor the status of Federal Reserve services and be alerted to issues via Slack.

## Installation

```bash
go install github.com/moov-io/frbstatus@latest
```

## Usage

```bash
./frbstatus                 # Show all services in table format
./frbstatus -unhealthy     # Only show unhealthy services
./frbstatus -format json   # Output in JSON format
```

Example CLI Usage:

```
$ frbstatus
```
```
FRB Service Status
==================

SERVICE                        STATUS
------------------------------ --------------------
Account Services               Normal Operations
Central Bank                   Normal Operations
Check 21                       Normal Operations
Check Adjustments              Normal Operations
FedACH                         ⚠️  Service Disruption
FedCash                        Normal Operations
FedNow                         Normal Operations
Fedwire Funds                  Normal Operations
Fedwire Securities             Normal Operations
National Settlement            Normal Operations
FedLine Advantage              Normal Operations
FedLine Command                Normal Operations
FedLine Direct                 Normal Operations
FedLine Web                    Normal Operations
FedMail                        Normal Operations
```

## Slack Alerts

Configure the Slack webhook URL to receive alerts for unhealthy services:

```bash
SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..." \
  ./frbstatus -unhealthy
```

When a Slack webhook URL is configured, the tool will:
- Detect unhealthy services (Service Issue or Service Disruption)
- Fetch details from the outage page
- Send a formatted message to Slack with:
  - Service name
  - Latest update timestamps
  - Link to view full details

## Flags

| Flag             | Description                                   |
|------------------|-----------------------------------------------|
| `-format string` | Output format: table or json (default: table) |
| `-unhealthy`     | Only report unhealthy services                |
