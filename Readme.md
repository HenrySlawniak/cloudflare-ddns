# cloudflare-ddns
_A cross-platform dynamic DNS client for Cloudflare_


### Usage

To use cloudflare-ddns, you need an API key and a domain on [Cloudflare](https://www.cloudflare.com). Generate an API key [here](https://www.cloudflare.com/a/account/my-account).

You can pass the configuration to cloudflare-ddns via command line flags, or environment variables. For more information on using command line flags, run with the `-h` flag.

The following environment variables are used to configure cloudflare-ddns:
 - CLOUDFLARE_DDNS_KEY `Your Cloudflare API key`
 - CLOUDFLARE_DDNS_EMAIL `Your Cloudflare API email`
 - CLOUDFLARE_DDNS_DOMAIN `The domain on Cloudflare to update`
 - CLOUDFLARE_DDNS_SUBDOMAIN `The subdomain to update, defaults to @`

### Determining External IP addresses

By default, cloudflare-ddns uses [ifcfg.org](https://ifcfg.org/) to determine external IP addresses.

[ifcfg.org](https://ifcfg.org/) is open source, and available [here](https://github.com/HenrySlawniak/ifcfg.org/)

You can optionally override this with the `-external-source` flag. For more information, run with the `-h` flag.
