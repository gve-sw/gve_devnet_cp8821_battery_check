# Cisco Phone 8821 Battery Check Utility

This code repository contains a script to fetch the HTTP status page from Cisco 8821 IP Phones.

This script will:
 - Take in a list of target IP addresses (Either a text list or CIDR range)
 - Query each IP & retrieve the HTTP status page
 - Parse HTML of each page & pull battery health/temperature info
 - Generate CSV reports for battery health

## Contacts
* Matt Schmitz (mattsc@cisco.com)

## Solution Components
* Cisco 8821 IP Phone

## Installation/Configuration

**[Step 1] Clone repo:**
```bash
git clone <repo_url>
```

**[Step 2] Install required dependencies:**

While in the module directory:
```bash
go get
```

**[Step 3] (OPTIONAL) Compile script:**

While the script can be run using `go run CP8821_battery_check.go <args>`, you can optionally compile using `go build CP8821_battery_check.go`.

The executable can then be distributed or installed without requiring a local Go installation.


## Usage

Run the script with **one** of the two following inputs:

 - `-infile` Provide an input text file of IP addresses to scan
 - `-cidr` Provide a CIDR range to scan

The following are additional optional arguments to modify the behavior of the script:

 - `-temp` Provide a temperature threshold for checking phone batteries, in Celsius. The default threshold is 50.0 C.
 - `-timeout` Provide a HTTP timeout for trying to reach an IP phone, in seconds. Default is 10 seconds.
 - `-v` Enable verbose logging, which will provide status on each IP phone as it is queried


**Note:** If using `-infile`, create a text file with each IP address to check on a new line, for example:

```text
10.10.10.10
20.20.20.20
30.30.30.30
40.40.40.40
```

**Note:** By default the script will try to use a TLS connection, but will fallback to HTTP.

# Screenshots

**Example of script execution:**

![/IMAGES/example_output.png](/IMAGES/example_output.png)

**Example of CSV report:**

![/IMAGES/example_report.png](/IMAGES/example_report.png)


### LICENSE

Provided under Cisco Sample Code License, for details see [LICENSE](LICENSE.md)

### CODE_OF_CONDUCT

Our code of conduct is available [here](CODE_OF_CONDUCT.md)

### CONTRIBUTING

See our contributing guidelines [here](CONTRIBUTING.md)

#### DISCLAIMER:
<b>Please note:</b> This script is meant for demo purposes only. All tools/ scripts in this repo are released for use "AS IS" without any warranties of any kind, including, but not limited to their installation, use, or performance. Any use of these scripts and tools is at your own risk. There is no guarantee that they have been through thorough testing in a comparable environment and we are not responsible for any damage or data loss incurred with their use.
You are responsible for reviewing and testing any scripts you run thoroughly before use in any non-testing environment.