# SqMGR - An online football squares manager application

## To start development

Get a working [Go](https://golang.org/doc/install) and [Docker](https://docs.docker.com/install/) environment setup. Then you can start the development server using the following:

```
$ make git-hooks   # install any necessary git-hooks
$ make dev-db      # starts a local PostgreSQL database in docker
$ make migrations  # runs db migrations
$ make run         # run the web server
```

Open your browser and navigate to [localhost:8080](http://localhost:8080).
