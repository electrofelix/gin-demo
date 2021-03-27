# gin-demo
Experimenting with gin

# Quickstart

Make sure you have docker installed with a version new enough for buildkit, along with docker-compose if you want to run the bundled helper file.
```bash
COMPOSE_DOCKER_CLI_BUILD=1 DOCKER_BUILDKIT=1 docker-compose up --build
```

To run the containers directly:
```bash
# launch the dynamodb container
docker run -p 8000:8000 -d --rm \
    amazon/dynamodb-local:1.15.0 \
        -XX:+UseContainerSupport -jar DynamoDBLocal.jar -sharedDb -inMemory

# build the image for the web app, should lint and test first
DOCKER_BUILDKIT=1 docker build -t electrofelix/gin-demo:latest .

# run the web-app and it should automatically create the needed table on launch
docker run --rm -it --net host --user $(id -u):$(id -g) electrofelix/gin-demo:latest
```
