![LayerG](.github/logo.png?raw=true "LayerG logo")
======


> Distributed server for social and realtime games and apps.

## Features

* **Users** - Register/login new users via social networks, email, or device ID.
* **Storage** - Store user records, settings, and other objects in collections.
* **Social** - Users can connect with friends, and join groups. Builtin social graph to see how users can be connected.
* **Chat** - 1-on-1, group, and global chat between users. Persist messages for chat history.
* **Multiplayer** - Realtime, or turn-based active and passive multiplayer.
* **Leaderboards** - Dynamic, seasonal, get top members, or members around a user. Have as many as you need.
* **Tournaments** - Invite players to compete together over prizes. Link many together to create leagues.
* **Parties** - Add team play to a game. Users can form a party and communicate with party members.
* **Purchase Validation** - Validate in-app purchases and subscriptions.
* **In-App Notifications** - Send messages and notifications to connected client sockets.
* **Runtime code** - Extend the server with custom logic written in Lua, TypeScript/JavaScript, or native Go code.
* **Matchmaker**, **dashboard**, **metrics**, and [more]().

Build scalable games and apps with a production ready server used by ambitious game studios and app developers [all around the world]().

## Getting Started

The server is simple to setup and run for local development and can be deployed to any cloud provider. See the [deployment notes](#deployment) for recommendations on how to deploy the project for production. LayerG server requires CockroachDB or another Postgres wire-compatible server as it's database.

### Docker

<img src="https://upload.wikimedia.org/wikipedia/commons/7/79/Docker_%28container_engine%29_logo.png" width="170">

The fastest way to run the server and the database is with Docker. Setup Docker and start the daemon.

1. Set up a [docker-compose file](https://github.com/u2u-labs/layerg-core/blob/main/docker-compose.yml) and place it in a folder for your project.

2. Run `docker-compose -f ./docker-compose.yml up` to download container images and run the servers.

### Binaries

You can run the servers with native binaries for your platform.


## Usage

LayerG supports a variety of protocols optimized for various gameplay or app use cases. For request/response it can use GRPC or the HTTP1.1+JSON fallback (REST). For realtime communication you can use WebSockets or rUDP.

For example with the REST API to authenticate a user account with a device identifier.

```shell
curl "127.0.0.1:7350/v2/account/authenticate/device?create=true" \
  --user "defaultkey:" \
  --data '{"id": "someuniqueidentifier"}'
```

Response:

> { <br>
>     "token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1MjQ5OTU2NDksInVpZCI6Ijk5Y2Q1YzUyLWE5ODgtNGI2NC04YThhLTVmMTM5YTg4MTgxMiIsInVzbiI6InhBb1RxTUVSdFgifQ.-3_rXNYx3Q4jKuS7RkxeMWBzMNAm0vl93QxzRI8p_IY" <br>
> }


## LayerG Console

The server provides a web UI which teams can use to inspect various data stored through the server APIs, view lightweight service metrics, manage player data, update storage objects, restrict access to production with permission profiles, and gain visibility into realtime features like active multiplayer matches. There is no separate installation required as it is embedded as part of the single server binary.

You can navigate to it on your browser on [http://127.0.0.1:7351](http://127.0.0.1:7351).

<img src=".github/accounts.jpg?raw=true" title="Account listing" width="1024" align="center">
<img src=".github/status.jpg?raw=true" title="Status view" width="125" align="left">
<img src=".github/storage.jpg?raw=true" title="Storage record view" width="125" align="left">
<img src=".github/match.jpg?raw=true" title="Running matches view" width="125" align="left">
<img src=".github/users.jpg?raw=true" title="Console users view" width="125" align="left">
<img src=".github/modules.jpg?raw=true" title="Loaded modules view" width="125">

## Deployment

LayerG can be deployed to any cloud provider such as Google Cloud, Azure, AWS, Digital Ocean, Heroku, or your own private cloud. You should setup and provision separate nodes for LayerG and CockroachDB.

The recommended minimum production infrastructure for CockroachDB is outlined in [these docs](https://www.cockroachlabs.com/docs/stable/recommended-production-settings.html#basic-hardware-recommendations) and LayerG can be run on instance types as small as "g1-small" on Google Cloud although we recommend a minimum of "n1-standard-1" in production. The specific hardware requirements will depend on what features of the server are used. Reach out to us for help and advice on what servers to run.

## Contribute

The development roadmap is managed as GitHub issues and pull requests are welcome. 

### Simple Builds

All dependencies required for a build are vendored as part of the Go project. We recommend a modern release of the Go toolchain and do not store the codebase in the old GOPATH.

1. Download the source tree.

   ```shell
   git clone "https://github.com/u2u-labs/layerg-core" layerg-core
   cd layerg-core
   ```

2. Build the project from source.

   ```shell
   go build -trimpath -mod=vendor
   ./layerg-core --version
   ```

### Full Source Builds

The codebase uses Protocol Buffers, GRPC, GRPC-Gateway, and the OpenAPI spec as part of the project. These dependencies are generated as sources and committed to the repository to simplify builds for contributors.

To build the codebase and generate all sources follow these steps.

1. Install the toolchain.

   ```shell
   go install \
       "google.golang.org/protobuf/cmd/protoc-gen-go" \
       "google.golang.org/grpc/cmd/protoc-gen-go-grpc" \
       "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway" \
       "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2"
   ```

2. Install the protobuf compiler

  ```shell
  brew install protobuf
  ```

3. If you've made changes to the embedded Console.

    ```shell
    cd console/ui
    ng serve
    ```

4. Re-generate the protocol buffers, gateway code and console UI.

   ```shell
   env PATH="$HOME/go/bin:$PATH" go generate -x ./...
   ```

5. Build the codebase.

   ```shell
   go build -trimpath -mod=vendor
   ```

### Testing

In order to run all the unit and integration tests run:

```shell
docker-compose -f ./docker-compose-tests.yml up --build --abort-on-container-exit; docker-compose -f ./docker-compose-tests.yml down -v
```

This will create an isolated environment with LayerG and database instances, run
all the tests, and drop the environment afterwards.

### License