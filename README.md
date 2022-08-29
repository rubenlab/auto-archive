# auto-archive

## Usage

`autoarchive config.yml`

## configuration

```
db: archive.db
server-name: "storage server"
root: /storage
scan-level: 3
scan-interval: 3
archive-interval: 30
email-to: tianming.yi@med.uni-goettingen.de
archive-command: "rm -rf \"${path}\""
backup-command: "cd ${dir} && tar --files-from=${file} --file=${id}/${date}/archive.tar"
notice-before:
  - 10
  - 5
  - 1
smtp-host: "smtp-mail.outlook.com"
smtp-port: 587
smtp-user: "rubsak1@outlook.com"
smtp-password: "efndkubpkaksfmsx"

```