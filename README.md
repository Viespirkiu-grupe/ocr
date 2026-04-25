# ocr


1. Copy [env.example](env.example) to `.env`

```shell
cp env.example .env
```

2. Set `API_KEY` in `.env`. The worker uses it as a bearer token when checking out tasks and submitting results.

```shell
API_KEY= # bearer token
GET_TASK_URL=https://viespirkiai.org/failas/ocr/checkout
POST_RESULT_URL=https://viespirkiai.org/failas/ocr/submit
BASE_FILE_URL=https://failai.viespirkiai.org/
CONCURRENCY=8 # 32 real cores + HT = floor(32/3) = 10 is value value you need to set
INBOX_DIR=/inbox
TESSERACT_LANG=lit+eng
```
3. Run the service

```shell
docker compose up -d
```

## Final notes

To stop the service use `docker compose down` or `docker-compose down`.

To rebuild the container, if you made code changes: `docker compose up -d --build` or `docker-compose up -d --build`.

`docker` can be easily replaced with `podman` in all of the examples above, if that is your jam. Both were tested and working.

## Get in touch

Exposing the service over the public internet is beyond the scope of this document, but do [reach out](https://viespirkiai.org/kontaktai) if you want to contribute a `golang` and need help.


# For advanced osx users

```
container system start
container build --tag ocr --file Dockerfile .
container run --rm -d -c [threads count] -m 1G --name ocr -v $(pwd)/inbox:/inbox --env-file=./.env -e API_KEY=... -e GET_TASK_URL=https://viespirkiai.org/failas/ocr/checkout -e POST_RESULT_URL=https://viespirkiai.org/failas/ocr/submit ocr
```
