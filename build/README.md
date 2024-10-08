Release Instructions
===

These instructions guide the release process for new official LayerG server builds.

## Steps

To build releases for a variety of platforms we use the excellent [xgo](https://github.com/techknowlogick/xgo) project. You will need Docker engine installed. These steps should be followed from the project root folder.

These steps are one off to install the required build utilities.

1. Install the xgo Docker image.

   ```
   docker pull techknowlogick/xgo:latest
   ```

2. Install the command line helper tool. Ensure "$GOPATH/bin" is on your system path to access the executable.

   ```
   go install src.techknowlogick.com/xgo@latest
   ```

These steps are run for each new release.

1. Update the CHANGELOG.

2. Add the CHANGELOG file and tag a commit.

   __Note__: In source control good semver suggests a "v" prefix on a version. It helps group release tags.

   ```
   git add CHANGELOG.md
   git commit -m "LayerG 2.1.0 release."
   git tag -a v2.1.0 -m "v2.1.0"
   git push origin v2.1.0 master
   ```

3. Execute the cross-compiled build helper.

   ```
   xgo --targets=darwin/arm64,darwin/amd64,linux/amd64,linux/arm64,windows/amd64 --trimpath --ldflags "-s -w -X main.version=2.1.0 -X main.commitID=$(git rev-parse --short HEAD 2>/dev/null)" github.com/u2u-labs/layerg-core
   ```

   This will build binaries for all target platforms supported officially by LayerG Labs.

4. Package up each release as a compressed bundle.

   ```
   tar -czf "layerg-<os>-<arch>.tar.gz" layerg README.md LICENSE CHANGELOG.md
   ```

5. Create a new draft release on GitHub and publish it with the compressed bundles.

## Build LayerG Image

With the release generated we can create the official container image.

1. Build the container image.

   ```
   cd build
   docker build "$PWD" --platform "linux/amd64" --file ./Dockerfile --build-arg commit="$(git rev-parse --short HEAD 2>/dev/null)" --build-arg version=2.1.0 -t layerglabs/layerg:2.1.0
   ```

2. Push the image to the container registry.

   ```
   docker tag <CONTAINERID> layerglabs/layerg:latest
   docker push layerglabs/layerg:2.1.0
   docker push layerglabs/layerg:latest
   ```

## Build LayerG Image (dSYM)

With the release generated we can also create an official container image which includes debug symbols.

1. Build the container image.

   ```
   cd build
   docker build "$PWD" --platform "linux/amd64" --file ./Dockerfile.dsym --build-arg commit="$(git rev-parse --short HEAD 2>/dev/null)" --build-arg version=2.1.0 -t layerglabs/layerg-dsym:2.1.0
   ```

2. Push the image to the container registry.

   ```
   docker tag <CONTAINERID> layerglabs/layerg-dsym:latest
   docker push layerglabs/layerg-dsym:2.1.0
   docker push layerglabs/layerg-dsym:latest
   ```

## Build Plugin Builder Image

With the official release image generated we can create a container image to help with Go runtime development.

1. Build the container image.

   ```
   cd build/pluginbuilder
   docker build "$PWD" --platform "linux/amd64" --file ./Dockerfile --build-arg commit="$(git rev-parse --short HEAD 2>/dev/null)" --build-arg version=2.1.0 -t layerglabs/layerg-pluginbuilder:2.1.0
   ```

2. Push the image to the container registry.

   ```
   docker tag <CONTAINERID> layerglabs/layerg-pluginbuilder:latest
   docker push layerglabs/layerg-pluginbuilder:2.1.0
   docker push layerglabs/layerg-pluginbuilder:latest
   ```
