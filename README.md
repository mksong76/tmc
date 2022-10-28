# Transmission Client(tmc)

```
# go install github.com/lanterrt/tmc@latest
```

# Configuration

Methods to set the configurations
1. parameter
2. environment variabl
3. configuration file ( `$HOME/.tmc/config.yml` )

## Required configuration

| Name     | Environment           | Description                   |
|----------|-----------------------|-------------------------------|
| host     | TRANSMISSION_HOST     | Host name transmission daemon |
| user     | TRANSMISSION_USER     | Name of the user              |
| password | TRANSMISSION_PASSWORD | Password of the user          |
| https    | TRANSMISSION_HTTPS    | `true` for using HTTPS        |
| port     | TRANSMISSION_PORT     | Port number for the server    |
| url      | TRANSMISSION_URL      | RPC URL(overriding others)    |

## Example

```
# tmc --host myhome.com --user myname --password "puhaha" ls
```

# Commands

## Save configuration to the file

```
# tmc --host myhome.com --user myname --password "puhaha" save
Save configuration to /Users/john/.tmc/config.yml
```

## Add a torrent

Add torrent with the file
```
# tmc add test.torrent
<ID>
```

Add torrent wth magnet URL
```
# tmc add "magnet://..."
<ID>
```

| Option   | Description                        |
|----------|------------------------------------|
| --detail | Print detail of added torrent      |
| --delete | Delete torrent file after addition |

## List torrents

```
# tmc ls
```

## Remove torrents

```
# tmc remove [a list of IDs]
<ID>
```

It removes specified torrents. If no torrents are specified, then it removes
all downloaded and stopped torrents.

| Option   | Description                        |
|----------|------------------------------------|
| --delete | Delete downloading files           |
