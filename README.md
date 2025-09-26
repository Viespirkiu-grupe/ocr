# ocr


1. Copy [env.example](env.example) to `.env`

```shell
cp env.example .env
```

2. Set `GET_TASK_URL`, `POST_RESULT_URL` and `CONCURRENCY` in `.env`

```shell
GET_TASK_URL= # tasks url  
POST_RESULT_URL= # results url
BASE_FILE_URL=https://failai-direct.viespirkiai.top/
CONCURRENCY=8 # 32 real cores + HT = 32/2 = 16 is value value you need to set
INBOX_DIR=./inbox
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

Exposing the service over the public internet is beyond the scope of this document, but do [reach out](https://viespirkiai.top/kontaktai) if you want to contribute a `golang` and need help.