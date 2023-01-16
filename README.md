# juicyrout

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/v-byte-cpu/juicyrout/blob/main/LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/v-byte-cpu/juicyrout/build.yml)](https://github.com/v-byte-cpu/juicyrout/actions/workflows/build.yml)
[![GoReportCard Status](https://goreportcard.com/badge/github.com/v-byte-cpu/juicyrout)](https://goreportcard.com/report/github.com/v-byte-cpu/juicyrout)

**juicyrout** is a man-in-the-middle attack reverse proxy designed for penetration testers to phish login credentials along with session cookies.

It was developed after experiencing the limitations of evilginx2. The main focus is on the following things:

* **lightweight**, it should be easy to set up without complicated phishlet configs
* **speed**, it must perform streaming processing of the response body instead of reading the whole response body into process memory (what happens in evilginx)
* **flexibility**, it should work with almost all sites out of the box with minimal configuration (e.g. just a list of session cookie names)

We use the following technical features to achieve this goal:
* only one wildcard tls certificate (no longer need to set up NS records to forward DNS servers to the proxy)
* js fetch/xhr hook (replace all dynamic URLs with the appropriate ones that point to the proxy)
* session handling on the server (authentication with specific lure URLs like in evilginx)
* cookie jar per session to persist all response cookies on the proxy and imitate legitimate browser session

## Quick Example

Let's imagine that you acquired a new DNS name, for example, `host.juicyrout` and you want to run a phishing instagram site 
on the domain `www.host.juicyrout` on port 8091. There are only three things you need to do:
  * generate a valid wildcard TLS certificate `*.host.juicyrout` e.g. with [let's encrypt](https://letsencrypt.org/docs/faq/#does-let-s-encrypt-issue-wildcard-certificates) 
  * add wildcard DNS A record for the domain `*.host.juicyrout`
  * run `juicyrout` with the desired config file

Create a file `config.yaml`:

```yaml
api_token: your_random_token_here
listen_addr: 0.0.0.0:8091
domain_name: host.juicyrout
external_port: 8091
tls_key: wildcard_host_juicyrout_key.pem
tls_cert: wildcard_host_juicyrout_cert.pem
phishlet_file: phishlets/instagram/config.yaml
domain_mappings:
  - proxy: www.host.juicyrout:8091
    target: www.instagram.com
```

* `api_token` is used for authentication in the admin REST API (see [api.go](https://github.com/v-byte-cpu/juicyrout/blob/main/api.go)). 
* `external_port` is used for incoming network traffic (in case you use Docker or run a reverse proxy in front of the juicyrout server, etc.)
* `phishlet_file` is a config file with phishlet (see [instagram phishlet](https://github.com/v-byte-cpu/juicyrout/blob/main/phishlets/instagram/config.yaml) for reference)
* `domain_mappings` describes a list of mappings between domain names with optional ports. For instance, 
`www.host.juicyrout:8091` with the specified configuration file will be mapped to `www.instagram.com` in http request 
and back in the response,
but `super.subdomain.instagram.com` (target address) will be mapped to `super-subdomain-instagram-com.host.juicyrout:8091` (proxy address)
by default.

For a complete list of configuration options, see [config.go](https://github.com/v-byte-cpu/juicyrout/blob/main/config.go).

Finally, run a phishing server:

```
juicyrout -c config.yaml
```

All captured credentials will by default be stored in the `creds.jsonl` file, and captured sessions will be stored in the 
`sessions.jsonl` file.

Enjoy!
